#!/bin/bash
dt=$(date +"%Y.%m.%d")
for idx in siem_alarms siem_events siem_alarm_events ossec suricata; do
  name=$idx-$dt
  text="resetting $name .."
  echo -e -n "\e[96m$text\e[0m\n"
  curl -fsS -X POST "localhost:9200/$name/_delete_by_query?pretty" -H 'Content-Type: application/json' \
  -d '{ "query": { "match_all": {} } }'
done

