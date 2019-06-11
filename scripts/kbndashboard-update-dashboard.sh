#!/bin/bash
temp=/tmp/dashboard-siem.json
target=./deployments/kibana/dashboard-siem.json
target_knql=./deployments/kibana/dashboard-siem_knql.json
source=$1

# make sure we're in root project dir
[[ ! $(git rev-parse --show-toplevel 2>/dev/null) = "$PWD" ]] && \
echo this script is intended for devs and should be called from the project root directory && \
exit 1

[ -z "$source" ] && echo need source kibana address as 1st argument && exit 1

./scripts/kbndashboard-export.sh $source $temp
[ "$?" != "0" ] && echo cannot export SIEM dashboard from $source && rm -rf $temp && exit 1

# fix the scripted field destination URL
sed -i 's/https:\/\/dsiem[^\/]*\//http:\/\/localhost:8080\//g' $temp

# make sure nested fields is enabled for KNQL version
if ! grep -q "\"nested\": true" $temp; then 
  echo make sure KNQL Kibana plugin is installed and nested fields is enabled for siem_alarms
  echo will skip KNQL dashboard version for now
else
  cp -r $temp $target_knql && echo $target_knql downloaded and patched successfully.
fi

# sed magic to remove "nested": false and keep the json intact
sed -i ':begin;$!N;s/,\n *\"nested\": false//;tbegin;P;D' $temp
sed -i ':begin;$!N;s/,\n *\"nested\": true//;tbegin;P;D' $temp

# remove kibana version
sed -i '0,/version/{/version/d;}' $temp
cp -r $temp $target && rm -rf $temp && echo $target downloaded and patched successfully.
rm -rf $temp