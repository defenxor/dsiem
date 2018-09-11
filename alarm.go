package main

import (
	"encoding/json"
	"os"
	"sync"
)

const (
	alarmLogs = "logs/siem_alarms.json"
)

var alarms siemAlarms
var alarmRemovalChannel chan removalChannelMsg

type alarm struct {
	ID          string      `json:"alarm_id"`
	Title       string      `json:"title"`
	Status      string      `json:"status"`
	Kingdom     string      `json:"kingdom"`
	Category    string      `json:"Category"`
	CreatedTime int64       `json:"created_time"`
	UpdateTime  int64       `json:"update_time"`
	Risk        int         `json:"risk"`
	RiskClass   string      `json:"risk_class"`
	Tag         string      `json:"tag"`
	SrcIPs      []string    `json:"src_ips"`
	DstIPs      []string    `json:"dst_ips"`
	Networks    []string    `json:"networks"`
	Rules       []alarmRule `json:"rules"`
}

type alarmRule struct {
	directiveRule
	EventCount int `json:"events_count"`
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
	a.Status = "Open"
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
	a.Tag = "Identified Threat"
	a.SrcIPs = b.SrcIPs
	a.DstIPs = b.DstIPs
	for i := range a.SrcIPs {
		a.Networks = append(a.Networks, getAssetNetworks(a.SrcIPs[i])...)
	}
	for i := range a.DstIPs {
		a.Networks = append(a.Networks, getAssetNetworks(a.DstIPs[i])...)
	}
	a.Networks = removeDuplicatesUnordered(a.Networks)
	a.Rules = []alarmRule{}
	for _, v := range b.Directive.Rules {
		rule := alarmRule{v, len(v.Events)}
		rule.Events = []string{} // so it will be omited during json marshaling
		a.Rules = append(a.Rules, rule)
	}

	err := a.updateElasticsearch(connID)
	if err != nil {
		logWarn("Alarm "+a.ID+" failed to update Elasticsearch! "+err.Error(), connID)
	}
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
	alarms.Alarms[idx] = alarms.Alarms[len(alarms.Alarms)-1]
	// write empty to last element
	alarms.Alarms[len(alarms.Alarms)-1] = alarm{}
	// truncate slice
	alarms.Alarms = alarms.Alarms[:len(alarms.Alarms)-1]
}
