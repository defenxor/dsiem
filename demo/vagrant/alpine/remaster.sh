#!/bin/sh

[ ! -e ./start-demo.sh ] && echo must be run from the script directory && exit 1
cp ./start-demo.sh /usr/local/bin/

# set /etc/resolv.conf if there's no name server
grep -q "nameserver" /etc/resolv.conf || echo nameserver 8.8.8.8 > /etc/resolv.conf

# requirements
apk add docker docker-compose git bash curl sudo || exit 1

rc-update add docker boot && \
service docker start

# clone or pull
[ -e /dsiem ] && cd /dsiem && git pull 
[ ! -e /dsiem ] && cd / && git clone https://github.com/defenxor/dsiem.git

# add user
deluser demo >/dev/null 2>&1
adduser -G docker -s /bin/bash demo 2>/dev/null
echo "demo ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/demo
echo "
/usr/local/bin/start-demo.sh
exit
" > /home/demo/.profile
chown -R demo /dsiem

rm -rf /etc/motd

# set login prompt
echo "Login as demo to start Dsiem demo script, or as root to do system administration.
both users have no password set by default.

Note that the VM needs to connect to the internet at least once to download dsiem tools and
docker images.
" > /etc/issue

# disable swap
sudo sed -i '/ swap / s/^/#/' /etc/fstab
swapoff -a

# set root and demo password to blank
echo demo:U6aMy0wojraho | sudo chpasswd -e 2>/dev/null
echo root:U6aMy0wojraho | sudo chpasswd -e 2>/dev/null
