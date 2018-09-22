package siem

import (
	"dsiem/internal/dsiem/pkg/asset"
	xc "dsiem/internal/dsiem/pkg/xcorrelator"
	"dsiem/internal/shared/pkg/fs"
	log "dsiem/internal/shared/pkg/logger"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/elastic/apm-agent-go"

	"github.com/spf13/viper"
)

const (
	alarmLogs = "siem_alarms.json"
)

var aLogFile string
var mediumRiskLowerBound int
var mediumRiskUpperBound int
var defaultTag string
var defaultStatus string
var alarmRemovalChannel chan removalChannelMsg
var privateIPBlocks []*net.IPNet

type alarm struct {
	sync.RWMutex
	ID              string           `json:"alarm_id"`
	Title           string           `json:"title"`
	Status          string           `json:"status"`
	Kingdom         string           `json:"kingdom"`
	Category        string           `json:"category"`
	CreatedTime     int64            `json:"created_time"`
	UpdateTime      int64            `json:"update_time"`
	Risk            int              `json:"risk"`
	RiskClass       string           `json:"risk_class"`
	Tag             string           `json:"tag"`
	SrcIPs          []string         `json:"src_ips"`
	DstIPs          []string         `json:"dst_ips"`
	ThreatIntels    []xc.IntelResult `json:"intel_hits,omitempty"`
	Vulnerabilities []xc.VulnResult  `json:"vulnerabilities,omitempty"`
	Networks        []string         `json:"networks"`
	Rules           []alarmRule      `json:"rules"`
}

type alarmRule struct {
	directiveRule
}

var alarms struct {
	sync.RWMutex
	al map[string]*alarm
}

// InitAlarm initialize alarm, storing result into logFile
func InitAlarm(logFile string) error {
	if err := fs.EnsureDir(path.Dir(logFile)); err != nil {
		return err
	}
	alarms.al = make(map[string]*alarm)

	mediumRiskLowerBound = viper.GetInt("medRiskMin")
	mediumRiskUpperBound = viper.GetInt("medRiskMax")
	defaultTag = viper.GetStringSlice("tags")[0]
	defaultStatus = viper.GetStringSlice("status")[0]

	if mediumRiskLowerBound < 2 || mediumRiskUpperBound > 9 ||
		mediumRiskLowerBound == mediumRiskUpperBound {
		return errors.New("Wrong value for medRiskMin or medRiskMax: " +
			"medRiskMax should be between 3-10, medRiskMin should be between 2-9, and medRiskMin should be < mdRiskMax")
	}

	aLogFile = logFile
	alarmRemovalChannel = make(chan removalChannelMsg)
	go func() {
		for {
			// handle incoming event, id should be the ID to remove
			m := <-alarmRemovalChannel
			go removeAlarm(m)
		}
	}()

	for _, cidr := range []string{
		"127.0.0.0/8",    // IPv4 loopback
		"10.0.0.0/8",     // RFC1918
		"172.16.0.0/12",  // RFC1918
		"192.168.0.0/16", // RFC1918
		"::1/128",        // IPv6 loopback
		"fe80::/10",      // IPv6 link-local
	} {
		_, block, _ := net.ParseCIDR(cidr)
		privateIPBlocks = append(privateIPBlocks, block)
	}

	return nil
}

func upsertAlarmFromBackLog(b *backLog, connID uint64, tx *elasticapm.Transaction) {
	defer elasticapm.DefaultTracer.Recover(tx)

	var a *alarm

	alarms.RLock()

	for _, v := range alarms.al {
		c := v
		if c.ID == b.ID {
			a = c
			break
		}
	}
	alarms.RUnlock()

	// if not found means new alarm
	if a == nil {
		alarms.Lock()
		newAlarm := alarm{}
		alarms.al[b.ID] = &newAlarm
		a = &newAlarm
		alarms.Unlock()
	}

	a.Lock()

	a.ID = b.ID
	a.Title = b.Directive.Name
	if a.Status == "" {
		a.Status = defaultStatus
	}
	if a.Tag == "" {
		a.Tag = defaultTag
	}

	a.Kingdom = b.Directive.Kingdom
	a.Category = b.Directive.Category
	if a.CreatedTime == 0 {
		a.CreatedTime = b.StatusTime
	}
	a.UpdateTime = b.StatusTime
	a.Risk = b.Risk
	switch {
	case a.Risk < mediumRiskLowerBound:
		a.RiskClass = "Low"
	case a.Risk >= mediumRiskLowerBound && a.Risk <= mediumRiskUpperBound:
		a.RiskClass = "Medium"
	case a.Risk > mediumRiskUpperBound:
		a.RiskClass = "High"
	}
	a.SrcIPs = b.SrcIPs
	a.DstIPs = b.DstIPs

	if xc.IntelEnabled {
		// do intel check in the background
		a.asyncIntelCheck(connID, tx)
	}

	if xc.VulnEnabled {
		// do vuln check in the background
		a.asyncVulnCheck(b, connID, tx)
	}

	for i := range a.SrcIPs {
		a.Networks = append(a.Networks, asset.GetAssetNetworks(a.SrcIPs[i])...)
	}
	for i := range a.DstIPs {
		a.Networks = append(a.Networks, asset.GetAssetNetworks(a.DstIPs[i])...)
	}
	a.Networks = removeDuplicatesUnordered(a.Networks)
	a.Rules = []alarmRule{}
	for _, v := range b.Directive.Rules {
		// rule := alarmRule{v, len(v.Events)}
		rule := alarmRule{v}
		rule.Events = []string{} // so it will be omited during json marshaling
		a.Rules = append(a.Rules, rule)
	}

	a.Unlock()

	err := a.updateElasticsearch(connID)
	if err != nil {
		tx.Result = "Alarm failed to update ES"
		a.RLock()
		log.Warn(log.M{Msg: "failed to update Elasticsearch! " + err.Error(), BId: a.ID, CId: connID})
		a.RUnlock()
		e := elasticapm.DefaultTracer.NewError(err)
		e.Transaction = tx
		e.Send()
	} else {
		tx.Result = "Alarm updated"
	}
}

func uniqStringSlice(cslist string) (result []string) {
	s := strings.Split(cslist, ",")
	result = removeDuplicatesUnordered(s)
	return
}

type vulnSearchTerm struct {
	ip   string
	port string
}

func sliceUniqMap(s []vulnSearchTerm) []vulnSearchTerm {
	seen := make(map[vulnSearchTerm]struct{}, len(s))
	j := 0
	for _, v := range s {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		s[j] = v
		j++
	}
	return s[:j]
}

func (a *alarm) asyncVulnCheck(b *backLog, connID uint64, tx *elasticapm.Transaction) {
	defer elasticapm.DefaultTracer.Recover(tx)

	go func() {
		// record prev value
		pVulnerabilities := a.Vulnerabilities

		// build IP:Port list
		terms := []vulnSearchTerm{}
		a.RLock()
		for _, v := range a.Rules {
			sIps := uniqStringSlice(v.From)
			ports := uniqStringSlice(v.PortFrom)
			sPort := strconv.Itoa(b.LastEvent.SrcPort)
			for _, z := range sIps {
				if z == "ANY" || z == "HOME_NET" || z == "!HOME_NET" || strings.Contains(z, "/") {
					continue
				}
				for _, y := range ports {
					if y == "ANY" {
						continue
					}
					terms = append(terms, vulnSearchTerm{z, y})
				}
				// also try to use port from last event
				if sPort != "0" {
					terms = append(terms, vulnSearchTerm{z, sPort})
				}
			}

			dIps := uniqStringSlice(v.To)
			ports = uniqStringSlice(v.PortTo)
			dPort := strconv.Itoa(b.LastEvent.DstPort)
			for _, z := range dIps {
				if z == "ANY" || z == "HOME_NET" || z == "!HOME_NET" || strings.Contains(z, "/") {
					continue
				}
				for _, y := range ports {
					if y == "ANY" {
						continue
					}
					terms = append(terms, vulnSearchTerm{z, y})
				}
				// also try to use port from last event
				if dPort != "0" {
					terms = append(terms, vulnSearchTerm{z, dPort})
				}
			}
		}
		a.RUnlock()

		terms = sliceUniqMap(terms)
		for i := range terms {
			log.Debug(log.M{Msg: "Evaluating " + terms[i].ip + ":" + terms[i].port, BId: a.ID, CId: connID})
			// skip existing entries
			alreadyExist := false
			a.RLock()
			for _, v := range a.Vulnerabilities {
				s := terms[i].ip + ":" + terms[i].port
				if v.Term == s {
					alreadyExist = true
					break
				}
			}
			a.RUnlock()
			if alreadyExist {
				log.Debug(log.M{Msg: "vuln checker: " + terms[i].ip + ":" + terms[i].port + " already exist", BId: a.ID, CId: connID})
				continue
			}

			p, err := strconv.Atoi(terms[i].port)
			if err != nil {
				continue
			}

			log.Debug(log.M{Msg: "actually checking vuln for " + terms[i].ip + ":" + terms[i].port, BId: a.ID, CId: connID})

			if found, res := xc.CheckVulnIPPort(terms[i].ip, p); found {
				a.Lock()
				a.Vulnerabilities = append(a.Vulnerabilities, res...)
				a.Unlock()
				log.Info(log.M{Msg: "found vulnerability for " + terms[i].ip + ":" + terms[i].port, CId: connID, BId: a.ID})
			}
		}

		// compare content of slice
		a.RLock()

		if reflect.DeepEqual(pVulnerabilities, a.Vulnerabilities) {
			a.RUnlock()
			return
		}
		a.RUnlock()
		err := a.updateElasticsearch(connID)
		if err != nil {
			a.RLock()
			log.Warn(log.M{Msg: "failed to update Elasticsearch after vulnerability check! " + err.Error(), BId: a.ID, CId: connID})
			a.RUnlock()
			e := elasticapm.DefaultTracer.NewError(err)
			e.Transaction = tx
			e.Send()
		}
	}()

}

func (a *alarm) asyncIntelCheck(connID uint64, tx *elasticapm.Transaction) {
	defer elasticapm.DefaultTracer.Recover(tx)
	go func() {

		IPIntel := a.ThreatIntels

		a.RLock()

		for i := range a.SrcIPs {
			// skip private IP
			if isPrivateIP(a.SrcIPs[i]) {
				continue
			}
			// skip existing entries
			alreadyExist := false
			for _, v := range a.ThreatIntels {
				if v.Term == a.SrcIPs[i] {
					alreadyExist = true
					break
				}
			}
			if alreadyExist {
				continue
			}

			if found, res := xc.CheckIntelIP(a.SrcIPs[i], connID); found {
				a.RUnlock()
				a.Lock()
				a.ThreatIntels = append(a.ThreatIntels, res...)
				a.Unlock()
				a.RLock()
				log.Info(log.M{Msg: "Found intel result for " + a.SrcIPs[i], CId: connID, BId: a.ID})

			}
		}

		for i := range a.DstIPs {
			// skip private IP
			if isPrivateIP(a.DstIPs[i]) {
				continue
			}
			// skip existing entries
			alreadyExist := false
			for _, v := range a.ThreatIntels {
				if v.Term == a.DstIPs[i] {
					alreadyExist = true
					break
				}
			}
			if alreadyExist {
				continue
			}
			if found, res := xc.CheckIntelIP(a.DstIPs[i], connID); found {
				a.RUnlock()
				a.Lock()
				a.ThreatIntels = append(a.ThreatIntels, res...)
				a.Unlock()
				a.RLock()

				log.Info(log.M{Msg: "Found intel result for " + a.DstIPs[i], CId: connID})
			}
		}

		// compare content of slice
		if reflect.DeepEqual(IPIntel, a.ThreatIntels) {
			a.RUnlock()
			return
		}

		a.RUnlock()

		err := a.updateElasticsearch(connID)
		if err != nil {
			a.RLock()
			log.Warn(log.M{Msg: "failed to update Elasticsearch after TI check! " + err.Error(), BId: a.ID, CId: connID})
			a.RUnlock()
			e := elasticapm.DefaultTracer.NewError(err)
			e.Transaction = tx
			e.Send()
		}
	}()

}

func (a *alarm) updateElasticsearch(connID uint64) error {
	a.RLock()
	log.Info(log.M{Msg: "updating Elasticsearch", BId: a.ID, CId: connID})

	aJSON, _ := json.Marshal(a)
	a.RUnlock()

	f, err := os.OpenFile(aLogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(string(aJSON) + "\n")
	return err
}

func removeAlarm(m removalChannelMsg) {
	alarms.Lock()
	log.Debug(log.M{Msg: "Lock obtained. Removing alarm", BId: m.ID, CId: m.connID})
	delete(alarms.al, m.ID)
	alarms.Unlock()
}

// to avoid copying mutex
func copyAlarm(dst *alarm, src *alarm) {
	dst.ID = src.ID
	dst.Title = src.Title
	dst.Status = src.Status
	dst.Kingdom = src.Kingdom
	dst.Category = src.Category
	dst.CreatedTime = src.CreatedTime
	dst.UpdateTime = src.UpdateTime
	dst.Risk = src.Risk
	dst.RiskClass = src.RiskClass
	dst.Tag = src.Tag
	dst.SrcIPs = src.SrcIPs
	dst.DstIPs = src.DstIPs
	dst.ThreatIntels = src.ThreatIntels
	dst.Vulnerabilities = src.Vulnerabilities
	dst.Networks = src.Networks
	dst.Rules = src.Rules
}

func isPrivateIP(ip string) bool {
	ipn := net.ParseIP(ip)
	for _, block := range privateIPBlocks {
		if block.Contains(ipn) {
			return true
		}
	}
	return false
}

func removeDuplicatesUnordered(elements []string) []string {
	encountered := map[string]bool{}

	// Create a map of all unique elements.
	for v := range elements {
		encountered[elements[v]] = true
	}

	// Place all keys from the map into a slice.
	result := []string{}
	for key := range encountered {
		result = append(result, key)
	}
	return result
}
