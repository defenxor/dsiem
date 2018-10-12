#!/bin/bash

rsync -avz --delete \
      --include "/web/dist" --exclude "/dsiem" --exclude "/ossimcnv" --exclude "/docs" --exclude "/nesd" --exclude "/dtester" --exclude "/temp" \
      --exclude "/logs" --exclude "/web/ui" --exclude "/web/ui-old" --exclude "/.git*" --exclude "/doc" --exclude "/test/pprof_results" \
      ./ mgmt184:/home/systemadm/dev/siem/src/dsiem/
pods=$(ssh mgmt184 -C "sudo kubectl get pods | grep dsiem | grep -v nats | grep -v apm | cut -d' ' -f1")
for p in $pods; do
ssh mgmt184 -C "sudo kubectl cp /home/systemadm/dev/siem/src/dsiem $p:/go/src/" &
done
