#!/bin/bash

# only do this script if $WSLHOST is defined. Otherwise just pause
if [ -z "$WSLHOST" ]; then
 [ "$1" == "test" ] && exit 0
 read -p "" && exit 0
fi

iface=$(ifconfig eth0 | grep 'inet ' | awk '{print $2}')

[ "$iface" == "" ] && echo cannot find eth0 IP address && exit 1

sshcmd="ssh -q -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null $WSLHOST"

$sshcmd hostname >/dev/null || { echo cannot connect to $WSLHOST using ssh; exit 1;}
[ "$1" == "test" ] && exit 0
$sshcmd -L 30001:$iface:8081 -nNT 

