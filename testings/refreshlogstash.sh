#!/bin/bash

rm -rf ./since_db.log.txt ./since_db.log.txt.1 ./since_db.log.txt.2
if grep -q "since_db.log.txt.1" ./siem.yml; then
 sed -i 's/since_db.log.txt.1/since_db.txt.2/g' ./siem.yml
else
 sed -i 's/since_db.log.txt.2/since_db.log.txt.1/g' ./siem.yml
fi
