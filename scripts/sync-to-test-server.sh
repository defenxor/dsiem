#!/bin/bash

rsync -avz --exclude "dsiem" --exclude "ossimconverter" \
      --exclude "logs" --exclude "web" --exclude "doc" --exclude "test" \
      ./* mgmt184:/home/systemadm/dev/siem/src/dsiem/
