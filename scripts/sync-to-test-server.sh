#!/bin/bash

rsync -avz  \
      --exclude "/dsiem" --exclude "/ossimcnv" --exclude "/nesd" \
      --exclude "logs" --exclude "web" --exclude "doc" \
      ./* mgmt184:/home/systemadm/dev/siem/src/dsiem/
