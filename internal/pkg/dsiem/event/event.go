// Copyright (c) 2018 PT Defender Nusa Semesta and contributors, All rights reserved.
//
// This file is part of Dsiem.
//
// Dsiem is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation version 3 of the License.
//
// Dsiem is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Dsiem. If not, see <https://www.gnu.org/licenses/>.

package event

import (
	"encoding/json"

	"github.com/defenxor/dsiem/internal/pkg/dsiem/asset"
)

// NormalizedEvent represents data received from logstash
type NormalizedEvent struct {
	ConnID       uint64 `json:"conn_id,omitempty"`
	EventID      string `json:"event_id"`
	Timestamp    string `json:"timestamp"`
	Title        string `json:"title,omitempty"`
	Sensor       string `json:"sensor"`
	PluginID     int    `json:"plugin_id,omitempty"`
	PluginSID    int    `json:"plugin_sid,omitempty"`
	Product      string `json:"product,omitempty"`
	Category     string `json:"category,omitempty"`
	SubCategory  string `json:"subcategory,omitempty"`
	SrcIP        string `json:"src_ip"`
	SrcPort      int    `json:"src_port"`
	DstIP        string `json:"dst_ip"`
	DstPort      int    `json:"dst_port"`
	Protocol     string `json:"protocol"`
	CustomData1  string `json:"custom_data1,omitempty"`
	CustomLabel1 string `json:"custom_label1,omitempty"`
	CustomData2  string `json:"custom_data2,omitempty"`
	CustomLabel2 string `json:"custom_label2,omitempty"`
	CustomData3  string `json:"custom_data3,omitempty"`
	CustomLabel3 string `json:"custom_label3,omitempty"`
	RcvdTime     int64  `json:"rcvd_time,omitempty"`    // for backpressure control
	TraceParent  string `json:"trace_parent,omitempty"` // for distributed tracing
	TraceState   string `json:"trace_state,omitempty"`  // for distributed tracing
}

// Channel define event channel with directive ID
type Channel struct {
	DirID int
	Ch    chan NormalizedEvent
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
