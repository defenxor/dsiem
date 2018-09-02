#!/bin/bash

rm -rf ./sincedb.txt ./sincedb.txt.1 ./sincedb.txt.2
if diff ./siem.yml ./siem1.yml >/dev/null 2>&1; then
  cp ./siem2.yml ./siem.yml 
else
  cp ./siem1.yml ./siem.yml
fi
