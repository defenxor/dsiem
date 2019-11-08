#!/bin/bash

# A kludgy hack to wait for external script to tell which interface to listen on.
# This is needed because:
# - No way to sniff traffic on a docker bridge
# - Docker env can vary, from dedicated server and Windows/WSL/OSX that are likely to have 
#   a network between victim and browser, and LInux users who likely have local only network
#   between them.
# - We only care about the interface that the victim is on

while true; do
  iface=$(cat /tmp/iface 2>/dev/null) && \
    chown -R suri /var/log/suricata && \
    /usr/bin/suricata -v -i ${iface} || echo $(date) "waiting for /tmp/iface to tell which interface to listen on .."
  sleep 3

done
