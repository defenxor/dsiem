#!/bin/bash
[ "$1" == "" ] || [ "$2" == "" ] && echo need the target kibana host as 1st argument, and dashboard json file output location as 2nd argument && exit 1

CURL="curl -s -f"

if [ ! -z "$ES_USERNAME" ] && [ ! -z "$ES_PASSWORD" ]; then
  echo "** using ES_USERNAME and ES_PASSWORD env variable for authentication **"
  CURL="curl -s -f -u $ES_USERNAME:$ES_PASSWORD"
fi

host=$1
echo downloading from kibana at http://${1}:5601 to $2 ..
curl -s -f -X GET "http://${1}:5601/api/kibana/dashboards/export?dashboard=87c18520-b337-11e8-b3e4-11404c6637fe" -H 'kbn-xsrf: true' >$2 &&
  echo done.
