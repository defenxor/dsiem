#!/bin/sh

[ ! -e ./start-demo.sh ] && echo must be run from the script directory && exit 1
cp ./start-demo.sh /usr/local/bin/

# requirements
apt-get install -y apt-transport-https ca-certificates curl software-properties-common git unzip && \
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add - >/dev/null 2>&1 && \
sudo add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" && \
sudo apt-get update && \
sudo apt-get install docker-ce -y 2>/dev/null

curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose && \
chmod +x /usr/local/bin/docker-compose

# clone or pull
[ -e /dsiem ] && cd /dsiem && git pull 
[ ! -e /dsiem ] && cd / && git clone https://github.com/defenxor/dsiem.git

# add user
deluser demo >/dev/null 2>&1
useradd -G docker -m -p U6aMy0wojraho -s /bin/bash demo && \
echo "demo ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/demo && \
echo "
/usr/local/bin/start-demo.sh
exit
" > /home/demo/.bashrc && \
chown -R demo /dsiem || exit 1

rm -rf /etc/motd

# set login prompt
echo "Login as demo to start Dsiem demo script, or as root to do system administration.
both users have no password set by default.

Note that the VM needs to connect to the internet at least once to download dsiem tools and
docker images.
" > /etc/issue

# set root and demo password to blank
echo root:U6aMy0wojraho | sudo chpasswd -e >/dev/null 2>&1
