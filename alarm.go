package main

import (
	"encoding/json"
	"net"
	"os"
	"reflect"
	"sync"
)

const (
	alarmLogs = "logs/siem_alarms.json"
)

var alarms siemAlarms
var alarmRemovalChannel chan removalChannelMsg
var privateIPBlocks []*net.IPNet

type alarm struct {
	ID          string        `json:"alarm_id"`
	Title       string        `json:"title"`
	Status      string        `json:"status"`
	Kingdom     string        `json:"kingdom"`
	Category    string        `json:"Category"`
	CreatedTime int64         `json:"created_time"`
	UpdateTime  int64         `json:"update_time"`
	Risk        int           `json:"risk"`
	RiskClass   string        `json:"risk_class"`
	Tag         string        `json:"tag"`
	SrcIPs      []string      `json:"src_ips"`
	SrcIPIntel  []intelResult `json:"src_ips_intel,omitempty"`
	DstIPs      []string      `json:"dst_ips"`
	DstIPIntel  []intelResult `json:"dst_ips_intel,omitempty"`
	Networks    []string      `json:"networks"`
	Rules       []alarmRule   `json:"rules"`
	mu          sync.RWMutex
}

type alarmRule struct {
	directiveRule
}

type siemAlarms struct {
	mu     sync.RWMutex
	Alarms []alarm `json:"alarm"`
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
	if intelEnabled {
		// do intel check in the background
		a.asyncIntelCheck(connID)
	}

	for i := range a.SrcIPs {
		a.Networks = append(a.Networks, getAssetNetworks(a.SrcIPs[i])...)
	}
	for i := range a.DstIPs {
		a.Networks = append(a.Networks, getAssetNetworks(a.DstIPs[i])...)
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
		logWarn("Alarm "+a.ID+" failed to update Elasticsearch! "+err.Error(), connID)
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
			if found, res := checkIntelIP(a.SrcIPs[i], connID); found {
				a.SrcIPIntel = append(a.SrcIPIntel, res...)
				logInfo("Found intel result for "+a.SrcIPs[i], connID)
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
			if found, res := checkIntelIP(a.DstIPs[i], connID); found {
				a.DstIPIntel = append(a.DstIPIntel, res...)
				logInfo("Found intel result for "+a.DstIPs[i], connID)
			}
		}
		// compare content of slice
		if reflect.DeepEqual(pSrcIPIntel, a.SrcIPIntel) && reflect.DeepEqual(pDstIPIntel, a.DstIPIntel) {
			return
		}
		err := a.updateElasticsearch(connID)
		if err != nil {
			logWarn("Alarm "+a.ID+" failed to update Elasticsearch after TI check! "+err.Error(), connID)
		}
	}()

}

func (a *alarm) updateElasticsearch(connID uint64) error {
	logInfo("alarm "+a.ID+" updating Elasticsearch.", connID)
	filename := progDir + "/" + alarmLogs
	aJSON, _ := json.Marshal(a)

	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(string(aJSON) + "\n")
	return err
}

func initAlarm() {
	alarmRemovalChannel = make(chan removalChannelMsg)
	go func() {
		for {
			// handle incoming event, id should be the ID to remove
			m := <-alarmRemovalChannel
			go removeAlarm(m)
		}
	}()

}

func removeAlarm(m removalChannelMsg) {
	logInfo("Trying to obtain write lock to remove alarm "+m.ID, m.connID)
	alarms.mu.Lock()
	defer alarms.mu.Unlock()
	logInfo("Lock obtained. Removing alarm "+m.ID, m.connID)
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
