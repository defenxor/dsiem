#!/bin/bash

es="localhost:9200"
template_loc="./docker/conf/logstash/index-template.d/es7/siem_alarms-template.json"
template_name="siem_alarms"
idx_name=siem_alarms-$(date +"%Y.%m.%d")
alias_name="siem_alarms_id_lookup"

echo "uploading index template $template_name .. " && \
curl -fsS -H "content-type:application/json" -XPUT "$es/_template/$template_name" -d@$template_loc >/dev/null && \
echo "creating index $idx_name .. " && \
# this may fail if already exist
curl -fsS -H "content-type:application/json" -XPUT "$es/$idx_name" >/dev/null 2>&1

echo "creating index alias $alias_name .. " && \
curl -fsS -H "content-type:application/json" -XPOST "$es/_aliases?pretty" \
-d '{ "actions" : [ { "add": { "index": "'$idx_name'", "alias": "'$alias_name'" }} ]}' >/dev/null && \
echo "done."



