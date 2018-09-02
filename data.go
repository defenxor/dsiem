package main

// { "timestamp": "2018-08-09T00:00:01Z", "sensor": "sensor1", "plugin_id": 1001, "plugin_sid": 2002,
// "priority": 3, "reliability": 1, "src_ip": "10.73.255.1", "src_port": "51231",
// "dst_ip": "10.73.255.10", "dst_port": 80, "protocol": "TCP", "userdata1": "ponda", "userdata2": "rossa" }

type (
	normalizedEvent struct {
		Timestamp    string `json:"timestamp"`
		Sensor       string `json:"sensor"`
		PluginID     int    `json:"plugin_id"`
		PluginSID    int    `json:"plugin_sid"`
		Priority     int    `json:"priority"`
		Reliability  int    `json:"reliability"`
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
)

func (e *normalizedEvent) valid() bool {
	if e.Timestamp == "" || e.Sensor == "" || e.PluginID == 0 || e.PluginSID == 0 {
		return false
	}
	return true
}
