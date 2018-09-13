#!/bin/bash

rsync -avz --exclude "siem" --exclude "ossim-directive-converter" --exclude "logs" --exclude "ui" --exclude "doc" --exclude "testings" ./* mgmt184:/home/systemadm/dev/siem/src/siem/
