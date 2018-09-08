package main

import (
	"encoding/json"
	"os"
)

const (
	alarmLogs = "logs/siem_alarms.json"
)

var alarms siemAlarms

type alarm struct {
	ID          string   `json:"alarm_id"`
	Title       string   `json:"title"`
	Status      string   `json:"status"`
	Kingdom     string   `json:"kingdom"`
	Category    string   `json:"Category"`
	CreatedTime int64    `json:"created_time"`
	UpdateTime  int64    `json:"update_time"`
	Risk        int      `json:"risk"`
	RiskClass   string   `json:"risk_class"`
	Tag         string   `json:"tag"`
	SrcIPs      []string `json:"src_ips"`
	DstIPs      []string `json:"dst_ips"`
	Networks    []string `json:"networks"`
}

type siemAlarms struct {
	Alarms []alarm `json:"alarm"`
}

func upsertAlarmFromBackLog(b *backLog) {
	var a *alarm
	for i := range alarms.Alarms {
		c := &alarms.Alarms[i]
		if c.ID == b.ID {
			a = &alarms.Alarms[i]
			break
		}
	}
	if a == nil {
		alarms.Alarms = append(alarms.Alarms, alarm{})
		a = &alarms.Alarms[len(alarms.Alarms)-1]
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

	err := a.updateElasticsearch()
	if err != nil {
		logger.Error("Alarm "+a.ID+" failed to update Elasticsearch! ", err)
	}
}

func (a *alarm) updateElasticsearch() error {
	logger.Info("alarm " + a.ID + " updating Elasticsearch.")
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
