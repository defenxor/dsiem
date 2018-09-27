#!/bin/bash
echo "" > logs/siem_alarms.json 
echo sending data to dsiem.
dt=$(date --utc --iso-8601=seconds)

random_string()
{
    cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w ${1:-32} | head -n 1
}
eid=$(random_string 4)

postdata=$(cat <<EOF
{ "timestamp": "${dt}", "event_id": "${eid}", "sensor": "sensor1", "plugin_id": 9002, "plugin_sid": 42, "priority": 3, "reliability": 1, "src_ip": "202.155.0.10", "src_port": 51231, "dst_ip": "10.23.51.67", "dst_port": 80, "protocol": "TCP", "custom_data1": "ponda", "custom_label1": "rossa" }
EOF
)

echo $postdata
curl -XPOST http://localhost:8080/events -d "$postdata"

