package main

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
	// fmt.Println(e.Timestamp, ":", e.Sensor, ":", e.PluginID, ":", e.PluginSID, ":", e.EventID, ":", e.SrcIP, ":", e.DstIP)
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
