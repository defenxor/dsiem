# Installation
 
The quickest and most reliable way to test Dsiem is to use the supplied Docker Compose files. They include Dsiem, all the required ELK stack, and an example log source (Suricata) preconfigured.

Then after you get a feel on how everything fits together, you can start integrating Dsiem into your existing or custom ELK deployment.

## Installing Dsiem

### Using Docker Compose

* Install <a href="https://git-scm.com/downloads">Git</a>, <a href="https://www.docker.com/get-started">Docker</a>, and 
  <a href="https://docs.docker.com/compose/install/">Docker Compose</a>.

* Clone this repository:

    ```shell
    git clone https://github.com/defenxor/dsiem
    ```

* Suricata needs to know which network interface to monitor traffic on. Tell it to use one of the active network interface on the host system like this (for `bash` shell):

    ```shell
    export PROMISC_INTERFACE=eth0
    ```
  
    Replace `eth0` above with the actual interface name. For testing purpose, it's not necessary to configure the interface to really operate in promiscuous mode.

* Run ELK, Suricata, and Dsiem in standalone mode:
  
    ```shell
    cd dsiem/deployments/docker && \
    docker-compose up
    ```

* Everything should be up and ready for testing in a few minutes.

* Here's things to note about the environment created by `docker-compose`:
  
    * Dsiem web UI should be accessible from http://localhost:8080/ui, Elasticsearch from http://localhost:9200, and Kibana from http://localhost:5601.
    * Suricata comes with <a href="https://rules.emergingthreats.net/open/suricata/rules/emerging-icmp_info.rules">Emerging Threats ICMP Info Ruleset</a> enabled, so you can easily trigger a test just by continuously pinging a host in the same subnet. Dsiem comes with an <a href="https://github.com/defenxor/dsiem/blob/master/configs/directives_dsiem-backend-0_testing2.json"> example directive configuration</a> that will intercept this "attack".
    * Recorded events will be stored in Elasticsearch index pattern `siem_events-*`, and alarms will be in `siem_alarms`. You can view their content from Kibana or the builtin SIEM web UI.

* Once Kibana is up at http://localhost:5601, you can import Dsiem dashboard and its dependencies using the following command:

    ```shell
    ./scripts/kbndashboard-import.sh localhost
    ```

### Using Existing ELK

* First make sure you already familiar with how Dsiem architecture works by testing it using the Docker Compose method above. Also note that these steps are only tested against ELK version 6.4.2, though it should work with any 6.x version with minor adjustment.

* Download Dsiem binary from the release page, unzip it, and run it on the target system, e.g. for Linux:

    ```shell

    [ "$EUID" -ne 0 ] && echo must be run as root! || (\
    export DSIEM_DIR=/var/dsiem && \
    mkdir -p $DSIEM_DIR && \
    curl https://github.com/defenxor/dsiem/releases/download/v0.1.0/dsiem-server-linux-amd64.zip -O /tmp/ && \
    unzip /tmp/dsiem-server-linux-amd64.zip -d $DSIEM_DIR && rm -rf /tmp/dsiem-server-linux-amd64.zip  && \
    cd $DSIEM_DIR && \
    ./dsiem serve)
    
    ```

* Install the following plugin to your Logstash instance:
    * logstash-filter-prune
    * logstash-filter-uuid

* Adjust and deploy the example configuration files for Logstash from <a href="https://github.com/defenxor/dsiem/tree/master/deployments/docker/conf/logstash">here</a>. Consult
  Logstash documentation if you have problem on this.

* Install filebeat on the same machine as dsiem, and configure it to use the provided example config file from <a href="https://github.com/defenxor/dsiem/tree/master/deployments/docker/conf/filebeat">here</a>.

    * Note that you should change `/var/log/dsiem` in that example to the `logs` directory inside dsiem install location (`/var/dsiem/logs` if using the above example).
  
    * Also make sure you adjust the logstash address variable inside `filebeat.yml` file to point to your Logstash endpoint address.

* Set Dsiem to auto-start by using something like this:
  
    ```shell

    [ "$EUID" -ne 0 ] && echo must be run as root! || ( \
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
    ./scripts/kbndashboard-import.sh ${your-kibana-IP-or-hostname}
    ```

## Uninstalling Dsiem

For `docker-compose` installation, just run the following:

```shell
cd dsiem/deployments/docker && \
docker-compose down -v
```
or
```shell
cd dsiem/deployments/docker && \
docker-compose -f docker-compose-cluster.yml down -v
```

For non `docker-compose` procedure, you will have to undo all the changes made manually, for example:

* Remove the extra logstash plugins and configuration files.
* Uninstall filebeat.
* Uninstall dsiem by deleting its directory and systemd unit file, if any.
