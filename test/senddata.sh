#!/bin/bash
echo "" > logs/siem_alarms.json 
echo sending data to dsiem.
curl -XPOST http://localhost:8080/events -d'
{ "@timestamp": "2018-08-09T00:00:01Z", "event_id": "asdad", "sensor": "sensor1", "plugin_id": 9002, "plugin_sid": 42, "priority": 3, "reliability": 1, "src_ip": "10.73.255.1", "src_port": 51231, "dst_ip": "103.254.148.124", "dst_port": 80, "protocol": "TCP", "custom_data1": "ponda", "custom_label1": "rossa" }
'
