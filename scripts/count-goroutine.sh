#!/bin/bash

curl http://10.73.184.90:8082/debug/vars/goroutine?debug=2 > /tmp/out.txt

cat /tmp/out.txt | grep "^dsiem" | sed 's/dsiem\/internal\/dsiem\/pkg//g'|grep -v vendor | cut -d'.' -f3-4 | sed 's/(.*//g' | sort | uniq -c | sed  "s/^[ \t]*//"

