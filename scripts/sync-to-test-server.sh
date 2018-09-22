#!/bin/bash

rsync -avz --delete \
      --include "/web/dist" --exclude "/dsiem" --exclude "/ossimcnv" --exclude "/nesd" \
      --exclude "/logs" --exclude "/web/ui" --exclude "/.git*" --exclude "/doc" --exclude "/test/pprof_results" \
      ./ mgmt184:/home/systemadm/dev/siem/src/dsiem/
ssh mgmt184 -C 'sudo kubectl cp /home/systemadm/dev/siem/src/dsiem dsiem-0:/go/src/'
