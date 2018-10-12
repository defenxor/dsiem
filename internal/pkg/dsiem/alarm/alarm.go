package alarm

import (
	"dsiem/internal/pkg/dsiem/asset"
	"dsiem/internal/pkg/dsiem/rule"
	xc "dsiem/internal/pkg/dsiem/xcorrelator"
	"dsiem/internal/pkg/shared/apm"
	"dsiem/internal/pkg/shared/fs"
	log "dsiem/internal/pkg/shared/logger"
	"encoding/json"
	"errors"
	"expvar"
	"net"
	"os"
	"path"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/jonhoo/drwmutex"

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
var alarmRemovalChannel chan string
var privateIPBlocks []*net.IPNet

var alarmCounter = expvar.NewInt("alarm_counter")

type alarm struct {
	// sync.RWMutex
	drwmutex.DRWMutex `json:"-"`
	ID                string           `json:"alarm_id"`
	Title             string           `json:"title"`
	Status            string           `json:"status"`
	Kingdom           string           `json:"kingdom"`
	Category          string           `json:"category"`
	CreatedTime       int64            `json:"created_time"`
	UpdateTime        int64            `json:"update_time"`
	Risk              int              `json:"risk"`
	RiskClass         string           `json:"risk_class"`
	Tag               string           `json:"tag"`
	SrcIPs            []string         `json:"src_ips"`
	DstIPs            []string         `json:"dst_ips"`
	ThreatIntels      []xc.IntelResult `json:"intel_hits,omitempty"`
	Vulnerabilities   []xc.VulnResult  `json:"vulnerabilities,omitempty"`
	Networks          []string         `json:"networks"`
	Rules             []alarmRule      `json:"rules"`
}

type alarmRule struct {
	rule.DirectiveRule
}

// alarms group all the alarm in a single collection
var alarms struct {
	drwmutex.DRWMutex
	al map[string]*alarm
}

func (a *alarm) asyncVulnCheck(srcPort, dstPort int, connID uint64, tx *elasticapm.Transaction) {
	if apm.Enabled() && tx != nil {
		defer elasticapm.DefaultTracer.Recover(tx)
	}

	go func() {
		// record prev value
		pVulnerabilities := a.Vulnerabilities

		// build IP:Port list
		terms := []vulnSearchTerm{}
		l := a.RLock()
		for _, v := range a.Rules {
			sIps := uniqStringSlice(v.From)
			ports := uniqStringSlice(v.PortFrom)
			sPort := strconv.Itoa(srcPort)
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
			dPort := strconv.Itoa(dstPort)
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
		l.Unlock()

		terms = sliceUniqMap(terms)
		for i := range terms {
			log.Debug(log.M{Msg: "Evaluating " + terms[i].ip + ":" + terms[i].port, BId: a.ID, CId: connID})
			// skip existing entries
			alreadyExist := false
			l := a.RLock()
			for _, v := range a.Vulnerabilities {
				s := terms[i].ip + ":" + terms[i].port
				if v.Term == s {
					alreadyExist = true
					break
				}
			}
			l.Unlock()
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
		l = a.RLock()

		if reflect.DeepEqual(pVulnerabilities, a.Vulnerabilities) {
			l.Unlock()
			return
		}
		err := a.updateElasticsearch(connID)
		l.Unlock()
		if err != nil {
			l := a.RLock()
			log.Warn(log.M{Msg: "failed to update Elasticsearch after vulnerability check! " + err.Error(), BId: a.ID, CId: connID})
			l.Unlock()
			if apm.Enabled() && tx != nil {
				e := elasticapm.DefaultTracer.NewError(err)
				e.Transaction = tx
				e.Send()
			}
		}
	}()

}

func (a *alarm) asyncIntelCheck(connID uint64, tx *elasticapm.Transaction) {
	if apm.Enabled() && tx != nil {
		defer elasticapm.DefaultTracer.Recover(tx)
	}

	go func() {

		IPIntel := a.ThreatIntels

		l := a.RLock()

		// loop over srcips and dstips
		p := append(a.SrcIPs, a.DstIPs...)
		p = removeDuplicatesUnordered(p)

		for i := range p {
			// skip private IP
			if isPrivateIP(p[i]) {
				continue
			}
			// skip existing entries
			alreadyExist := false
			for _, v := range a.ThreatIntels {
				if v.Term == p[i] {
					alreadyExist = true
					break
				}
			}
			if alreadyExist {
				continue
			}
			if found, res := xc.CheckIntelIP(p[i], connID); found {
				l.Unlock()
				a.Lock()
				a.ThreatIntels = append(a.ThreatIntels, res...)
				a.Unlock()
				l = a.RLock()
				log.Info(log.M{Msg: "Found intel result for " + p[i], CId: connID, BId: a.ID})
			}
		}

		// compare content of slice
		if reflect.DeepEqual(IPIntel, a.ThreatIntels) {
			l.Unlock()
			return
		}

		err := a.updateElasticsearch(connID)
		l.Unlock()
		if err != nil {
			l := a.RLock()
			log.Warn(log.M{Msg: "failed to update Elasticsearch after TI check! " + err.Error(), BId: a.ID, CId: connID})
			l.Unlock()
			if apm.Enabled() && tx != nil {
				e := elasticapm.DefaultTracer.NewError(err)
				e.Transaction = tx
				e.Send()
			}
		}
	}()

}

func (a alarm) updateElasticsearch(connID uint64) error {
	log.Info(log.M{Msg: "alarm updating Elasticsearch", BId: a.ID, CId: connID})
	aJSON, _ := json.Marshal(a)

	f, err := os.OpenFile(aLogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	f.SetDeadline(time.Now().Add(60 * time.Second))

	_, err = f.WriteString(string(aJSON) + "\n")
	return err
}

func removeAlarm(id string) {
	alarms.Lock()
	log.Debug(log.M{Msg: "Lock obtained. Removing alarm", BId: id})
	delete(alarms.al, id)
	alarms.Unlock()
}

// UpdateCount set and return the count of alarms
func UpdateCount() (count int) {
	l := alarms.RLock()
	count = len(alarms.al)
	l.Unlock()
	alarmCounter.Set(int64(count))
	return
}

// RemovalChannel returns the channel used to send alarm ID to delete
func RemovalChannel() chan string {
	return alarmRemovalChannel
}

func removalListener() {
	go func() {
		for {
			// handle incoming event, id should be the ID to remove
			m := <-alarmRemovalChannel
			go removeAlarm(m)
		}
	}()
}

// Init initialize alarm, storing result into logFile
func Init(logFile string) error {
	if err := fs.EnsureDir(path.Dir(logFile)); err != nil {
		return err
	}
	alarms.DRWMutex = drwmutex.New()
	alarms.Lock()
	alarms.al = make(map[string]*alarm)
	alarms.Unlock()

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
	alarmRemovalChannel = make(chan string)
	removalListener()

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

func findOrCreateAlarm(id string) (a *alarm) {
	l := alarms.RLock()
	for k := range alarms.al {
		if alarms.al[k].ID == id {
			a = alarms.al[id]
			l.Unlock()
			return
		}
	}
	l.Unlock()
	alarms.Lock()
	alarms.al[id] = &alarm{}
	alarms.al[id].DRWMutex = drwmutex.New()
	alarms.al[id].ID = id
	a = alarms.al[id]
	alarms.Unlock()
	return
}

// Upsert creates or update alarms
// backlog struct is decomposed here to avoid circular dependency
func Upsert(id, name, kingdom, category string,
	srcIPs, dstIPs []string, lastSrcPort, lastDstPort, risk int, statusTime int64,
	rules []rule.DirectiveRule, connID uint64,
	tx *elasticapm.Transaction) {

	if apm.Enabled() {
		defer elasticapm.DefaultTracer.Recover(tx)
	}

	a := findOrCreateAlarm(id)
	a.Lock()

	a.Title = name
	if a.Status == "" {
		a.Status = defaultStatus
	}
	if a.Tag == "" {
		a.Tag = defaultTag
	}

	a.Kingdom = kingdom
	a.Category = category
	if a.CreatedTime == 0 {
		a.CreatedTime = statusTime
	}
	a.UpdateTime = statusTime
	a.Risk = risk
	switch {
	case a.Risk < mediumRiskLowerBound:
		a.RiskClass = "Low"
	case a.Risk >= mediumRiskLowerBound && a.Risk <= mediumRiskUpperBound:
		a.RiskClass = "Medium"
	case a.Risk > mediumRiskUpperBound:
		a.RiskClass = "High"
	}
	a.SrcIPs = srcIPs
	a.DstIPs = dstIPs

	// only do these if tx is not nil, if it is that means this is just a timeout signal update
	// from backlog ticker
	if xc.IntelEnabled && tx != nil {
		// do intel check in the background
		a.asyncIntelCheck(connID, tx)
	}

	if xc.VulnEnabled && tx != nil {
		// do vuln check in the background
		a.asyncVulnCheck(lastSrcPort, lastDstPort, connID, tx)
	}

	for i := range a.SrcIPs {
		a.Networks = append(a.Networks, asset.GetAssetNetworks(a.SrcIPs[i])...)
	}
	for i := range a.DstIPs {
		a.Networks = append(a.Networks, asset.GetAssetNetworks(a.DstIPs[i])...)
	}
	a.Networks = removeDuplicatesUnordered(a.Networks)
	a.Rules = []alarmRule{}
	for _, v := range rules {
		// rule := alarmRule{v, len(v.Events)}
		rule := alarmRule{v}
		rule.Events = []string{} // so it will be omited during json marshaling
		rule.StickyDiff = ""
		a.Rules = append(a.Rules, rule)
	}

	err := a.updateElasticsearch(connID)
	a.Unlock()
	if err != nil {
		tx.Result = "Alarm failed to update ES"
		l := a.RLock()
		log.Warn(log.M{Msg: "failed to update Elasticsearch! " + err.Error(), BId: a.ID, CId: connID})
		l.Unlock()
		if apm.Enabled() && tx != nil {
			e := elasticapm.DefaultTracer.NewError(err)
			e.Transaction = tx
			e.Send()
		}
	} else {
		if apm.Enabled() && tx != nil {
			tx.Result = "Alarm updated"
		}
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
