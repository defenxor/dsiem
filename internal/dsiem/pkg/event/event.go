package event

import (
	"dsiem/internal/dsiem/pkg/asset"
	"encoding/json"
)

// NormalizedEvent represents data received from logstash
type NormalizedEvent struct {
	ConnID       uint64 `gojay:"conn_id,omitempty"`
	EventID      string `gojay:"event_id"`
	Timestamp    string `gojay:"timestamp"`
	Sensor       string `gojay:"sensor"`
	PluginID     int    `gojay:"plugin_id,omitempty"`
	PluginSID    int    `gojay:"plugin_sid,omitempty"`
	Product      string `gojay:"product,omitempty"`
	Category     string `gojay:"category,omitempty"`
	SubCategory  string `gojay:"subcategory,omitempty"`
	SrcIP        string `gojay:"src_ip"`
	SrcPort      int    `gojay:"src_port"`
	DstIP        string `gojay:"dst_ip"`
	DstPort      int    `gojay:"dst_port"`
	Protocol     string `gojay:"protocol"`
	CustomData1  string `gojay:"custom_data1,omitempty"`
	CustomLabel1 string `gojay:"custom_label1,omitempty"`
	CustomData2  string `gojay:"custom_data2,omitempty"`
	CustomLabel2 string `gojay:"custom_label2,omitempty"`
	CustomData3  string `gojay:"custom_data3,omitempty"`
	CustomLabel3 string `gojay:"custom_label3,omitempty"`
	RcvdTime     int64  `gojay:"rcvd_time,omitempty"` // for backpressure control
}

// Valid check if event contains valid content for required fields
func (e *NormalizedEvent) Valid() bool {
	// fmt.Println(e.Timestamp, ":", e.Sensor, ":", e.EventID, ":", e.SrcIP, ":", e.DstIP, ":", e.PluginID, ":", e.PluginSID)
	if e.Timestamp == "" || e.Sensor == "" || e.EventID == "" || e.SrcIP == "" || e.DstIP == "" {
		return false
	}

	if e.PluginID == 0 || e.PluginSID == 0 {
		if e.Product == "" || e.Category == "" {
			return false
		}
	}
	return true
}

// FromBytes initialize NormalizedEvent
func (e *NormalizedEvent) FromBytes(b []byte) error {
	err := json.Unmarshal(b, &e)
	return err
}

// ToBytes return byte rep of event
func (e *NormalizedEvent) ToBytes() (b []byte, err error) {
	b, err = json.Marshal(e)
	return
}

// SrcIPInHomeNet check if event SrcIP is is HOME_NET
func (e *NormalizedEvent) SrcIPInHomeNet() bool {
	res, _ := asset.IsInHomeNet(e.SrcIP)
	return res
}

// DstIPInHomeNet check if event DstIP is is HOME_NET
func (e *NormalizedEvent) DstIPInHomeNet() bool {
	res, _ := asset.IsInHomeNet(e.DstIP)
	return res
}
