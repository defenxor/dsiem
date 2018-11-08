#!/bin/bash

curl -X GET "http://kibana.appl184.mss.defenxor.com:5601/api/kibana/dashboards/export?dashboard=87c18520-b337-11e8-b3e4-11404c6637fe" -H 'kbn-xsrf: true' > ./deployments/kibana/dashboard-siem.json

