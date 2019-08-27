# Installation
 
The quickest and most reliable way to test Dsiem is to use the supplied Docker Compose files. They include Dsiem, all the required ELK stack, and an example log source (Suricata) pre-configured.

Then after you get a feel on how everything fits together, you can start integrating Dsiem into your existing or custom ELK deployment.

## Installing Dsiem

### Using Docker Compose

* Install [Docker](https://www.docker.com/get-started), and [Docker Compose](https://docs.docker.com/compose/install/).
* Installing Docker and Docker Compose on Centos 7
    ``` 
    $ sudo yum -y update
    $ sudo reboot
    $ sudo sed -i s/^SELINUX=.*$/SELINUX=permissive/ /etc/selinux/config
    $ sudo setenforce 0
    $ sudo yum install -y yum-utils device-mapper-persistent-data lvm2 unzip wget curl git
    $ sudo yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
    $ sudo yum install docker-ce
    $ sudo usermod -aG docker $(whoami)
    $ sudo systemctl enable docker.service
    $ sudo systemctl start docker.service
    $ sudo yum install epel-release
    $ sudo yum install -y python-pip
    $ sudo pip install docker-compose
    $ sudo pip install --upgrade pip
    $ docker-compose version
    ```


* Copy this repository from [here](https://github.com/defenxor/dsiem/archive/master.zip), unzip it, then open the result in terminal.

    ```shell
    $ wget https://github.com/defenxor/dsiem/archive/master.zip
    $ unzip master.zip && cd dsiem-master/
    ```

* Suricata needs to know which network interface to monitor traffic on. Tell it to use the network interface that has a working Internet connection on your system like this (for `bash` shell):

    ```shell
    $ export PROMISC_INTERFACE=eth0
    ```
  
    Replace `eth0` above with the actual interface name given by `ifconfig` or similar commands. For testing purpose, it's not necessary to configure the interface to really operate in promiscuous mode.
    When you installing on Virtual Machine running on top of VMware Workstation please follow [This link](https://isc.sans.edu/forums/diary/Running+Snort+on+VMWare+ESXi/15899/), or [This link](https://kb.vmware.com/s/article/1004099) when you running on top of ESXI/Vsphere to enable `Promiscuous mode`

* Set the owner of filebeat config file to root ([here's why](https://www.elastic.co/guide/en/beats/libbeat/6.4/config-file-permissions.html)):
    ```shell
    $ cd deployments/docker && \
    sudo chown root conf/filebeat/filebeat.yml
    ```

* Run ELK, Suricata, and Dsiem in standalone mode:
  
    ```shell
    $ docker-compose pull
    $ docker-compose up
    ```

* Everything should be up and ready for testing in a few minutes. Here's things to note about the environment created by `docker-compose`:
  
    * Dsiem web UI should be accessible from http://localhost:8080/ui, Elasticsearch from http://localhost:9200, and Kibana from http://localhost:5601.
    * Suricata comes with [Emerging Threats ICMP Info Ruleset](https://rules.emergingthreats.net/open/suricata/rules/emerging-icmp_info.rules) enabled and `EXTERNAL_NET: "any"`, so you can easily trigger a test alarm just by continuously pinging a host in the same subnet. Dsiem comes with an [example directive configuration](https://github.com/defenxor/dsiem/blob/master/configs/directives_dsiem-backend-0_testing1.json) that will intercept this "attack".
    * Recorded events will be stored in Elasticsearch index pattern `siem_events-*`, and alarms will be in `siem_alarms`. You can view their content from Kibana or the builtin SIEM web UI.

#### Importing Kibana Dashboard

* Once Kibana is up at http://localhost:5601, you can import Dsiem dashboard and its dependencies using the following command:

    ```shell
    $ ./scripts/kbndashboard-import.sh localhost ./deployments/kibana/dashboard-siem.json
    ```
  Do notice that like any Kibana dashboard, Dsiem dashboard also expect the underlying indices (in this case `siem_alarms` and `siem_events-*`) to have been created before it can be accessed without error. This means you will need to trigger the test alarm described above before attempting to use the dashboard.
  
### Using Existing ELK

* First make sure you're already familiar with how Dsiem architecture works by testing it using the Docker Compose method above. Also note that these steps are only tested against ELK version 6.4.2 and 6.8.0, though it should work with any 6.x version (or likely 7.x as well) with minor adjustment.

* Download Dsiem binary from the release page, unzip it, and run it on the target system, e.g. for Linux (please use dsiem latest version accordingly for the download URL):

    ```shell

    $ [ "$EUID" -ne 0 ] && echo must be run as root! || (\
    export DSIEM_DIR=/var/dsiem && \
    mkdir -p $DSIEM_DIR && \
    curl https://github.com/defenxor/dsiem/releases/download/v0.1.0/dsiem-server-linux-amd64.zip -O /tmp/ && \
    unzip /tmp/dsiem-server-linux-amd64.zip -d $DSIEM_DIR && rm -rf /tmp/dsiem-server-linux-amd64.zip  && \
    cd $DSIEM_DIR && \
    ./dsiem serve)
    
    ```

* Install the following plugin to your Logstash instance:
    * [logstash-filter-prune](https://www.elastic.co/guide/en/logstash/current/plugins-filters-prune.html)
    * [logstash-filter-uuid](https://www.elastic.co/guide/en/logstash/current/plugins-filters-uuid.html)

* Adjust and deploy the example configuration files for Logstash from [here](https://github.com/defenxor/dsiem/tree/master/deployments/docker/conf/logstash). Consult
  Logstash documentation if you have problem on this.

* Install Filebeat on the same machine as dsiem, and configure it to use the provided example config file from [here](https://github.com/defenxor/dsiem/tree/master/deployments/docker/conf/filebeat).

    * Note that you should change `/var/log/dsiem` in that example to the `logs` directory inside dsiem install location (`/var/dsiem/logs` if using the above example).
  
    * Also make sure you adjust the logstash address variable inside `filebeat.yml` file to point to your Logstash endpoint address.

* Set Dsiem to auto-start by using something like this (for systemd-based Linux):
  
    ```shell

    $ [ "$EUID" -ne 0 ] && echo must be run as root! || ( \
    cat <<EOF > /etc/systemd/system/dsiem.service 
    [Unit]
    Description=Dsiem
    After=network.target

    [Service]
    Type=simple
    WorkingDirectory=/var/dsiem
    ExecStart=/var/dsiem/dsiem serve
    Restart=on-failure

    [Install]
    WantedBy=multi-user.target
    EOF
    systemctl daemon-reload && \
    systemctl enable dsiem.service && \
    systemctl start dsiem.service && \
    systemctl status dsiem.service)
    ```
* Dsiem web UI should be accessible from http://HostIPAddress:8080/ui

* Import Kibana dashboard from `deployments/kibana/dashboard-siem.json`. This step will also install all Kibana index-patterns (`siem_alarms` and `siem_events`) that will be linked to from Dsiem web UI.

    ```shell
    $ ./scripts/kbndashboard-import.sh ${your-kibana-IP-or-hostname} ./deployments/kibana/dashboard-siem.json
    ```

## Uninstalling Dsiem

For `docker-compose` installation, just run the following:

```shell
$ cd dsiem/deployments/docker && \
docker-compose down -v
```
or
```shell
$ cd dsiem/deployments/docker && \
docker-compose -f docker-compose-cluster.yml down -v
```

For non `docker-compose` procedure, you will have to undo all the changes made manually, for example:

* Remove the extra logstash plugins and configuration files.
* Uninstall Filebeat.
* Uninstall Dsiem by deleting its directory and systemd unit file, if any.
