package main

// { "timestamp": "2018-08-09T00:00:01Z", "sensor": "sensor1", "plugin_id": 1001, "plugin_sid": 2002,
// "priority": 3, "reliability": 1, "src_ip": "10.73.255.1", "src_port": "51231",
// "dst_ip": "10.73.255.10", "dst_port": 80, "protocol": "TCP", "userdata1": "ponda", "userdata2": "rossa" }

import (
	"encoding/json"
)

type normalizedEvent struct {
	ConnID       uint64
	EventID      string `json:"event_id"`
	Timestamp    string `json:"@timestamp"`
	Sensor       string `json:"sensor"`
	PluginID     int    `json:"plugin_id"`
	PluginSID    int    `json:"plugin_sid"`
	Reliability  int    `json:"reliability"`
	Priority     int    `json:"priority"`
	SrcIP        string `json:"src_ip"`
	SrcPort      int    `json:"src_port"`
	DstIP        string `json:"dst_ip"`
	DstPort      int    `json:"dst_port"`
	Protocol     string `json:"protocol"`
	CustomData1  string `json:"custom_data1"`
	CustomLabel1 string `json:"custom_label1"`
	CustomData2  string `json:"custom_data2"`
	CustomLabel2 string `json:"custom_label2"`
	CustomData3  string `json:"custom_data3"`
	CustomLabel3 string `json:"custom_label3"`
}

func (e *normalizedEvent) valid() bool {
	if e.Timestamp == "" || e.Sensor == "" || e.PluginID == 0 || e.PluginSID == 0 || e.EventID == "" ||
		e.SrcIP == "" || e.DstIP == "" {
		return false
	}
	return true
}

func (e *normalizedEvent) fromBytes(b []byte) error {
	err := json.Unmarshal(b, &e)
	return err
}

func (e *normalizedEvent) srcIPInHomeNet() bool {
	res, _ := isInHomeNet(e.SrcIP)
	return res
}

func (e *normalizedEvent) dstIPInHomeNet() bool {
	res, _ := isInHomeNet(e.DstIP)
	return res
}

type directiveRule struct {
	Name        string   `json:"name"`
	Stage       int      `json:"stage"`
	PluginID    int      `json:"plugin_id"`
	PluginSID   []int    `json:"plugin_sid"`
	Occurrence  int      `json:"occurrence"`
	From        string   `json:"from"`
	To          string   `json:"to"`
	PortFrom    string   `json:"port_from"`
	PortTo      string   `json:"port_to"`
	Protocol    string   `json:"protocol"`
	Reliability int      `json:"reliability"`
	Timeout     int64    `json:"timeout"`
	StartTime   int64    `json:"start_time"`
	Events      []string `json:"events,omitempty"`
}

type directive struct {
	ID       int             `json:"id"`
	Name     string          `json:"name"`
	Priority int             `json:"priority"`
	Kingdom  string          `json:"kingdom"`
	Category string          `json:"category"`
	Rules    []directiveRule `json:"rules"`
}

type directives struct {
	Directives []directive `json:"directives"`
}
