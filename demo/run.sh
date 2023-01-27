#!/bin/bash

scriptdir="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
cd $scriptdir

clear
title() {
  echo -e -n "\e[96m$@\e[0m"
}

for c in docker docker-compose curl sudo rm sleep grep awk dirname ip; do
  command -v $c >/dev/null || {
    echo -e "\ncannot find a required command: $c"
    exit 1
  }
done

OP=$1

function end_demo() {
  cd $scriptdir/docker &&
    DEMO_HOST=$DEMO_HOST PROMISC_INTERFACE=$PROMISC_INTERFACE docker-compose down -v
  exit $?
}

[ "$OP" == "down" ] && end_demo
[ "$OP" == "pull" ] && {
  cd docker && docker-compose pull
  exit $?
}

title "** DSIEM DEMO **\n\n"

DEMO_HOST=${DEMO_HOST:-$2}
iface=${iface:-$3}

function list_include_item() {
  local list="$1"
  local item="$2"
  if [[ $list =~ (^|[[:space:]])"$item"($|[[:space:]]) ]]; then
    # yes, list include item
    result=0
  else
    result=1
  fi
  return $result
}

if [ "$DEMO_HOST" == "" ] || [ "$iface" == "" ]; then
  ifaces=$(ip a | grep -E '^[0-9]:*' | cut -d' ' -f2 | cut -d: -f1 | grep -v lo | xargs)
  [ "$ifaces" == "" ] && echo cannot get list of network interfaces && exit 1
  while ! list_include_item "$ifaces" "$iface"; do
    read -p "Which network interface will be accessible by your browser? [$ifaces]: " iface
  done
  ifaceIP=$(ip a | grep $iface | grep inet | awk '{ print $2}' | cut -d'/' -f1)
  [ "$ifaceIP" == "" ] && echo cannot get interface $iface IP address && exit 1
  while [ "$DEMO_HOST" == "" ]; do
    read -p "What is the hostname/IP that your browser will use to access the demo web interface? [$ifaceIP]: " DEMO_HOST
    DEMO_HOST=${DEMO_HOST:-$ifaceIP}
  done
  echo "
**
The demo web interface will be setup to listen on $iface ($ifaceIP), and you will use
http://$DEMO_HOST:8000 to access it.

Any network address translation and routing in-between this docker server and your browser must 
allow the above to happen.

Press any key to confirm the above and continue, or CTRL-C to abort.
**"
  read -p ""
else
  ifaceIP=$(ip a | grep $iface | grep inet | awk '{ print $2}' | cut -d'/' -f1)
fi

trap end_demo INT

title "** making sure beat config files are owned by root .. "
sudo chown root $(find ./docker/conf/filebeat ./docker/conf/filebeat-es/ ./docker/conf/auditbeat/ -name "*.yml") ||
  {
    echo cannot set filebeat config owner to root
    exit 1
  }
echo done

cd docker && DEMO_HOST=$DEMO_HOST docker-compose up -d || exit $?
cd ..

# first find target internal IP
title "** finding target IP address .. "
while [ "$targetIP" == "" ]; do
  targetIP=$(docker exec -it logstash ping -c1 shellshock | grep "PING shellshock" | cut -d' ' -f3 | sed 's/(//;s/)//;') ||
    {
      echo cannot determine target IP address
      end_demo
    }
  sleep 3
done
echo done

nesfile="./docker/conf/nesd/csv/nessus_shellshock.csv"
mkdir -p $(dirname $nesfile)
title "** preparing nesd CSV file .. "

rm -rf $nesfile
./scripts/nesd-upsert-csv.sh $targetIP 80 $nesfile >/dev/null
while ! docker restart dsiem-nesd >/dev/null; do
  sleep 3
done
echo done

# wise readiness
title "** verifying $ifaceIP in Wise .. "
while ! (curl -fsS localhost:8083/ip/$ifaceIP 2>&1 | grep -q 'testing only'); do
  sleep 1
done
echo done

# nesd readiness
title "** verifying $targetIP:80 in Nesd .. "
./scripts/nesd-upsert-csv.sh $targetIP 80 ./docker/conf/nesd/csv/nessus_shellshock.csv
while ! (curl -fsS 'localhost:8082/?ip='$targetIP'&port=80' 2>&1 | grep -q 'CVE-2014-6271'); do
  sleep 1
done
echo done

# elasticsearch readiness
title "** ensuring elasticsearch is ready .. "
while ! curl -fsS localhost:9200 >/dev/null 2>&1; do
  sleep 1
done
echo done

title "** preparing es indices .. "
./scripts/es-prepare.sh >/dev/null || exit 1
echo done

# prep suricata
title "** setting up suricata interface .. "
targetif=$(docker exec -it shellshock ip a | grep 'eth0@' | cut -d: -f1) || {
  echo "cannot get shellshock container interface!"
  end_demo
}
surif=$(docker exec -it suricata ip a | grep if${targetif} | cut -d: -f2 | cut -d'@' -f1) || {
  echo "cannot get suricata interface!"
  end_demo
}
docker exec suricata bash -c "echo $surif > /tmp/iface" || {
  echo "cannot set suricata interface"
  end_demo
}
sleep 3
docker exec suricata ps axuw | grep -q suricata || {
  echo "cannot find suricata process inside its container!"
  end_demo
}
echo done

# target readiness
title "** making sure target is ready .. "
while ! $(curl -fsS localhost:8081/cgi-bin/vulnerable 2>/dev/null | grep -q average); do
  docker restart shellshock >/dev/null >&1
  sleep 3
done
echo done

# logstash readiness
title "** ensuring logstash is ready .. "
while ! docker exec logstash curl localhost:9600 >/dev/null 2>&1; do
  sleep 1
done
echo done

# filebeat-es readiness
title "** ensuring filebeat-es index template is correctly installed .. "
while true; do
  docker exec filebeat-es /usr/share/filebeat/filebeat setup --index-management >/dev/null 2>&1
  $(curl -fsS 'localhost:9200/_template/filebeat*' | grep -q dsiem) && break
  sleep 1
done
echo done

title "** ensuring filebeat-es uses the correct mapping .. "
docker restart filebeat-es >/dev/null 2>&1
sleep 3
curl -fsS -XDELETE 'localhost:9200/filebeat-*/' >/dev/null 2>&1
docker restart filebeat-es >/dev/null 2>&1
echo done

# ossec-syslog core dumped if the destination address uses hostname, likely chroot problem

logstashIP=$(docker exec ossec ping -c1 logstash | grep "PING logstash" | cut -d' ' -f3 | sed 's/(//;s/)//;') ||
  {
    echo cannot obtain logstash IP address
    end_demo
  }
title "** setting ossec syslog destination to logstash IP ($logstashIP) .. "
docker exec ossec /usr/bin/sed -i s/logstash/$logstashIP/g /var/ossec/etc/ossec.conf ||
  {
    echo cannot replace ossec syslog destination
    exit 1
  }
docker exec ossec /var/ossec/bin/ossec-control restart >/dev/null 2>&1 ||
  {
    echo cannot restart ossec
    exit 1
  }
sleep 3
docker exec ossec bash -c "ps axuw | grep -v grep | grep -q syslog" ||
  {
    echo cannot find ossec syslog process
    exit 1
  }
echo done

title "** ossec initialization .. "
while ! docker logs ossec 2>&1 | grep -q 'ossec-syscheckd: INFO: Ending syscheck scan (forwarding database).'; do
  sleep 1
done
echo done

title "** ossec integrity check logging .. "
while ! docker exec ossec grep -q md5 /var/ossec/logs/alerts/alerts.log; do
  docker exec ossec bash -c 'echo $RANDOM >> /var/www/html/test.html'
  sleep 1
done
echo done

# index readiness

for i in suricata ossec; do
  title "** waiting for $i index to become available .. "
  while ! $(curl -s -X GET "localhost:9200/_cat/indices?v&pretty" | grep -q "$i"); do
    [ "$i" == "ossec" ] && docker exec ossec bash -c 'echo $RANDOM >> /var/www/html/test.html'
    # generate curl traffic to trigger suricata signature
    [ "$i" == "suricata" ] && curl $ifaceIP:8081 >/dev/null 2>&1
    sleep 1
  done
  echo done
done

# kibana readiness
title "** waiting kibana to become ready .. "
while ! $(curl -s localhost:5601/app/kibana | grep -q "content security policy"); do
  sleep 1
done
echo done

title "** installing kibana dashboards .. "
cp -r ./kibana /tmp/ &&
  sed -i "s/localhost/$DEMO_HOST/g" /tmp/kibana/dashboard-siem.json
while (./scripts/kbndashboard-import.sh localhost /tmp/kibana/dashboard-siem.json | grep -q "failed"); do
  sleep 1
done
echo done
rm -rf /tmp/kibana

title "** installing additional kibana index patterns .. "
while (./scripts/idxpattern-import.sh localhost ./kibana/dashboard-siem.json | grep -q "failed"); do
  sleep 1
done
echo done

title "** removing test entries .. "
idxname=$(curl -fsS "localhost:9200/_cat/indices?v&pretty" | grep siem_events | awk '{ print $3}')
curl -fsS -X POST "localhost:9200/$idxname/_delete_by_query?pretty" -H 'Content-type:application/json' -d'
{ "query": { "match": { "plugin_id": 50001 } } }' >/dev/null 2>&1 &&
  curl -fsS -X DELETE "localhost:9200/auditbeat-*/" >/dev/null 2>&1 &&
  echo done

title "** ensuring dsiem-demo-frontend readiness  .. "
docker start dsiem-demo-frontend >/dev/null 2>&1 &&
  echo done

echo "
Demo is ready, access the web interface from http://${DEMO_HOST}:8000/

(Press CTRL-C to tear down the demo and exit)
"
while true; do sleep 600; done
