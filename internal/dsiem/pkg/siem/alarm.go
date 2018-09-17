package siem

import (
	"dsiem/internal/dsiem/pkg/asset"
	log "dsiem/internal/dsiem/pkg/logger"
	xc "dsiem/internal/dsiem/pkg/xcorrelator"
	"dsiem/internal/shared/pkg/fs"
	"encoding/json"
	"net"
	"os"
	"path"
	"reflect"
	"sync"
)

const (
	alarmLogs = "siem_alarms.json"
)

var aLogFile string
var alarms siemAlarms
var alarmRemovalChannel chan removalChannelMsg
var privateIPBlocks []*net.IPNet

type alarm struct {
	ID          string           `json:"alarm_id"`
	Title       string           `json:"title"`
	Status      string           `json:"status"`
	Kingdom     string           `json:"kingdom"`
	Category    string           `json:"Category"`
	CreatedTime int64            `json:"created_time"`
	UpdateTime  int64            `json:"update_time"`
	Risk        int              `json:"risk"`
	RiskClass   string           `json:"risk_class"`
	Tag         string           `json:"tag"`
	SrcIPs      []string         `json:"src_ips"`
	SrcIPIntel  []xc.IntelResult `json:"src_ips_intel,omitempty"`
	DstIPs      []string         `json:"dst_ips"`
	DstIPIntel  []xc.IntelResult `json:"dst_ips_intel,omitempty"`
	Networks    []string         `json:"networks"`
	Rules       []alarmRule      `json:"rules"`
	mu          sync.RWMutex
}

type alarmRule struct {
	directiveRule
}

type siemAlarms struct {
	mu     sync.RWMutex
	Alarms []alarm `json:"alarm"`
}

// InitAlarm initialize alarm, storing result into logFile
func InitAlarm(logFile string) error {
	if err := fs.EnsureDir(path.Dir(logFile)); err != nil {
		return err
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

func upsertAlarmFromBackLog(b *backLog, connID uint64) {
	var a *alarm

	for i := range alarms.Alarms {
		c := &alarms.Alarms[i]
		if c.ID == b.ID {
			a = &alarms.Alarms[i]
			break
		}
	}
	if a == nil {
		alarms.mu.Lock()
		alarms.Alarms = append(alarms.Alarms, alarm{})
		a = &alarms.Alarms[len(alarms.Alarms)-1]
		alarms.mu.Unlock()
	}
	a.ID = b.ID
	a.Title = b.Directive.Name
	if a.Status == "" {
		a.Status = "Open"
	}
	if a.Tag == "" {
		a.Tag = "Identified Threat"
	}

	a.Kingdom = b.Directive.Kingdom
	a.Category = b.Directive.Category
	if a.CreatedTime == 0 {
		a.CreatedTime = b.StatusTime
	}
	a.UpdateTime = b.StatusTime
	a.Risk = b.Risk
	switch {
	case a.Risk <= 2:
		a.RiskClass = "Low"
	case a.Risk >= 3 && a.Risk <= 6:
		a.RiskClass = "Medium"
	case a.Risk >= 7:
		a.RiskClass = "High"
	}
	a.SrcIPs = b.SrcIPs
	a.DstIPs = b.DstIPs
	if xc.IntelEnabled {
		// do intel check in the background
		a.asyncIntelCheck(connID)
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

	err := a.updateElasticsearch(connID)
	if err != nil {
		log.Warn("Alarm "+a.ID+" failed to update Elasticsearch! "+err.Error(), connID)
	}
}

func (a *alarm) asyncIntelCheck(connID uint64) {
	go func() {
		// lock to make sure the alreadyExist test is useful
		a.mu.Lock()
		defer a.mu.Unlock()

		pSrcIPIntel := a.SrcIPIntel
		pDstIPIntel := a.DstIPIntel

		for i := range a.SrcIPs {
			// skip private IP
			if isPrivateIP(a.SrcIPs[i]) {
				continue
			}
			// skip existing entries
			alreadyExist := false
			for _, v := range a.SrcIPIntel {
				if v.Term == a.SrcIPs[i] {
					alreadyExist = true
					break
				}
			}
			if alreadyExist {
				continue
			}
			if found, res := xc.CheckIntelIP(a.SrcIPs[i], connID); found {
				a.SrcIPIntel = append(a.SrcIPIntel, res...)
				log.Info("Found intel result for "+a.SrcIPs[i], connID)
			}
		}
		for i := range a.DstIPs {
			// skip private IP
			if isPrivateIP(a.DstIPs[i]) {
				continue
			}
			// skip existing entries
			alreadyExist := false
			for _, v := range a.DstIPIntel {
				if v.Term == a.DstIPs[i] {
					alreadyExist = true
					break
				}
			}
			if alreadyExist {
				continue
			}
			if found, res := xc.CheckIntelIP(a.DstIPs[i], connID); found {
				a.DstIPIntel = append(a.DstIPIntel, res...)
				log.Info("Found intel result for "+a.DstIPs[i], connID)
			}
		}
		// compare content of slice
		if reflect.DeepEqual(pSrcIPIntel, a.SrcIPIntel) && reflect.DeepEqual(pDstIPIntel, a.DstIPIntel) {
			return
		}
		err := a.updateElasticsearch(connID)
		if err != nil {
			log.Warn("Alarm "+a.ID+" failed to update Elasticsearch after TI check! "+err.Error(), connID)
		}
	}()

}

func (a *alarm) updateElasticsearch(connID uint64) error {
	log.Info("alarm "+a.ID+" updating Elasticsearch.", connID)
	aJSON, _ := json.Marshal(a)

	f, err := os.OpenFile(aLogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(string(aJSON) + "\n")
	return err
}

func removeAlarm(m removalChannelMsg) {
	log.Info("Trying to obtain write lock to remove alarm "+m.ID, m.connID)
	alarms.mu.Lock()
	defer alarms.mu.Unlock()
	log.Info("Lock obtained. Removing alarm "+m.ID, m.connID)
	idx := -1
	for i := range alarms.Alarms {
		if alarms.Alarms[i].ID == m.ID {
			idx = i
		}
	}
	if idx == -1 {
		return
	}
	// copy last element to idx location
	alarms.Alarms[len(alarms.Alarms)-1].mu.Lock()
	alarms.Alarms[idx].mu.Lock()
	copyAlarm(&alarms.Alarms[idx], &alarms.Alarms[len(alarms.Alarms)-1])
	alarms.Alarms[idx].mu.Unlock()
	alarms.Alarms[len(alarms.Alarms)-1].mu.Unlock()

	// write empty to last element
	alarms.Alarms[len(alarms.Alarms)-1] = alarm{}
	// truncate slice
	alarms.Alarms = alarms.Alarms[:len(alarms.Alarms)-1]
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
	dst.SrcIPIntel = src.SrcIPIntel
	dst.DstIPs = src.DstIPs
	dst.DstIPIntel = src.DstIPIntel
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
