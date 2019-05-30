#!/bin/bash
[ "$1" == "" ] || [ "$2" == "" ] && echo need the target kibana host as 1st argument, and dashboard json file output location as 2nd argument && exit 1

host=$1
echo downloading from kibana at http://${1}:5601 to $2 ..
curl -s -f -X GET "http://${1}:5601/api/kibana/dashboards/export?dashboard=87c18520-b337-11e8-b3e4-11404c6637fe" -H 'kbn-xsrf: true' > $2 && \
echo done.

