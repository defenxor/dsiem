#!/bin/bash

# set nameserver if it isn't set by dhcp
grep -q "nameserver" /etc/resolv.conf || sudo sh -c "echo nameserver 8.8.8.8 > /etc/resolv.conf"

# get the main eth iface first, there should only be one
[ -e /home/vagrant ] && defeth=$(ip route | grep default | cut -d' ' -f5)
eth=$(ip a | grep ^[0-9] | grep -vE "(br|veth|docker|lo|$defeth)"| cut -d: -f2)
n=$(echo "$eth" | wc -l)
[ "$n" != "1" ] && echo cannot find a single main network interface on this system! && read -p "" && exit 1
eth=$(echo $eth)
ethip=$(ip a | grep $eth | grep inet | awk '{ print $2}' | cut -d/ -f1)
echo "using $eth ($ethip) ..."

# download dpluger and the rest
echo "downloading dsiem tools latest version (if internet is available) .."
cd /usr/local/bin && \
sudo wget https://github.com/defenxor/dsiem/releases/latest/download/dsiem-tools_linux_amd64.zip -O tmp.zip && \
sudo unzip -o tmp.zip && \
sudo rm -rf tmp.zip

cd /dsiem/demo
echo "pulling dsiem source changes (if internet is available) .."
git pull >/dev/null 2>&1
echo "pulling docker images as needed (if internet is available) .."
DEMO_HOST=$ethip ./run.sh pull
./run.sh up $ethip $eth

