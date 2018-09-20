#!/bin/bash
echo "" > logs/siem_alarms.json 
echo sending data to dsiem.
dt=$(date --utc --iso-8601=seconds)

postdata=$(cat <<EOF
{ "@timestamp": "${dt}", "event_id": "asdad", "sensor": "sensor1", "plugin_id": 9002, "plugin_sid": 42, "priority": 3, "reliability": 1, "src_ip": "10.73.255.1", "src_port": 51231, "dst_ip": "10.23.51.67", "dst_port": 80, "protocol": "TCP", "custom_data1": "ponda", "custom_label1": "rossa" }
EOF
)
curl -XPOST http://localhost:8080/events -d "$postdata"

