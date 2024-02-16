#!/bin/bash

host=$1
dname=$(dirname $2)
for f in $(ls $dname/idxpattern-*); do
  name=$(echo $f | cut -d'-' -f2 | cut -d. -f1)
  [ "$name" == "siem_alarm_events" ] && continue
  echo -n "Installing index pattern $f .. "
  idxpattern=$(cat $f)
  res=$(curl -fsS -o /dev/null -X POST "http://${host}:5601/api/saved_objects/index-pattern/$name-*" -H 'kbn-xsrf: true' -H 'Content-Type: application/json' -d "$idxpattern" 2>&1)
  (echo $res | grep -q "409" && echo done) || echo "failed to install: $res"

done
