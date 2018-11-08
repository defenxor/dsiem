#!/bin/bash

[ "$1" == "" ] && echo need the target kibana host as 1st argument && exit 1

host=$1
dashboard=$(cat ./deployments/kibana/dashboard-siem.json)
if [ "$?" == "0" ]; then
  curl -s -S -X POST "http://${host}:5601/api/kibana/dashboards/import" -H 'kbn-xsrf: true' -H 'Content-Type: application/json' -d "$dashboard" -o /dev/null && \
  curl -XPOST -H "Content-Type: application/json" -H "kbn-xsrf: true" ${host}:5601/api/kibana/settings/defaultIndex -d '{"value": "siem_alarms"}' && \
  echo "" && echo dashboard installed successfully
fi
