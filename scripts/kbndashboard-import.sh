#!/bin/bash

[ "$1" == "" ] || [ "$2" == "" ] || [ ! -e $2 ] && echo need the target kibana host as 1st argument, and dashboard json file to upload as 2nd argument && exit 1

host=$1
dashboard=$(cat $2)
command -v curl >/dev/null 2>&1 || { echo the required curl command is not available && exit 1 ;}

while ! curl -fsS ${host}:5601 -o /dev/null; do
  echo cannot connect to ${host}:5601, will retry in 5 sec ..
  sleep 5
done

echo -n installing kibana dashboard to ${host}:5601 .. && \
curl -fsS -o /dev/null -X POST "http://${host}:5601/api/kibana/dashboards/import?force=true" -H 'kbn-xsrf: true' -H 'Content-Type: application/json' -d "$dashboard"  && \
echo done. && \
echo -n setting default index to siem_alarms .. && \
curl -fsS -o /dev/null -XPOST -H "Content-Type: application/json" -H "kbn-xsrf: true" ${host}:5601/api/kibana/settings/defaultIndex -d '{"value": "siem_alarms"}'  && \
echo done && \
echo dashboard installed successfully.
