# Dsiem 

[![CircleCI](https://circleci.com/gh/defenxor/dsiem.svg?style=shield&circle-token=def79b85071ad74a4bb86fd9d225bb09d00694c5)](https://circleci.com/gh/defenxor/dsiem) [![Go Report Card](https://goreportcard.com/badge/github.com/defenxor/dsiem)](https://goreportcard.com/report/github.com/defenxor/dsiem) [![Coverage Status](https://coveralls.io/repos/github/defenxor/dsiem/badge.svg?branch=master&t=4EXv3N)](https://coveralls.io/github/defenxor/dsiem?branch=master) [![License: GPL v3](https://img.shields.io/badge/License-GPL%20v3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0) 

Dsiem is a security event correlation engine for [ELK stack](https://www.elastic.co/elk-stack), allowing the platform to be used as a dedicated and full-featured [SIEM](https://en.wikipedia.org/wiki/Security_information_and_event_management) system.

Dsiem provides [OSSIM](https://www.alienvault.com/products/ossim)-style correlation for normalized logs/events, perform lookup/query to threat intelligence and vulnerability information sources, and produces risk-adjusted alarms.

![Example Kibana Dashboard](/docs/images/kbn-dashboard.png)

## Features

* Runs in standalone or clustered mode with [NATS](https://nats.io/) as messaging bus between frontend and backend nodes. Along with ELK, this made the entire SIEM platform horizontally scalable.
* OSSIM-style correlation and directive rules, bridging easier transition from OSSIM.
* Alarms enrichment with data from threat intel and vulnerability information sources. Builtin support for [Moloch Wise](https://github.com/aol/moloch/wiki/WISE) (which supports Alienvault OTX and others) and Nessus CSV exports, with support for other sources can easily be implemented as plugins.
* Instrumentation supported through metricbeat and/or Elastic APM server. No need for extra stack for this purpose.
* Builtin rate and backpressure control, set the minimum and maximum events/second (EPS) received from Logstash depending on your hardware capacity and acceptable delays in event processing.
* Loosely coupled, designed to be composable with other infrastructure platform, and doesnt try to do everything. As an example, there's no authentication support by design, since implementing that using nginx or other frontend should provide better security. Loose coupling  also means that it's possible to use Dsiem as a correlation engine with non ELK stack if needed.
* Batteries included:
    * A directive conversion tool that reads OSSIM XML directive file and translate it to Dsiem JSON-style config.
    * A SIEM plugin creator tool that will read off an existing index pattern from Elasticsearch, and creates the necessary Logstash configuration to clone the relevant fields' content to Dsiem.
    * A helper tool to serve Nessus CSV files over the network to Dsiem.
    * A minimalistic Angular web UI just for basic alarms management (closing, tagging), and easy pivoting to the relevant indices in Kibana to perform the actual analysis.
* Obviously a cloud-native, twelve-factor app, and all that jazz.

## How It Works

![Simple Architecture](/docs/images/simple-arch.png)

On the diagram above:

* Log sources send their logs to syslog/filebeat, which then send it to Logstash with a unique identifying field.

* Logstash parses the logs using different filters based on the log sources type, and send the results to Elasticsearch, typically creating a single index pattern for each log type (e.g. `suricata-*` for logs received from Suricata IDS, `ssh-*` for SSH logs, etc.). 

* The above is a common pattern used for monitoring logs with ELK stack, so dsiem is preconfigured to integrate with that kind of scenario.

* Dsiem uses a special purpose logstash config file to clone incoming event from log sources, right after logstash has done parsing it. Through the same config file, the new cloned event is processed (independently from the original event) to collect Dsiem required fields like Title, Source IP, Destination IP, and so on.

  *This special logstash config file in Dsiem can be thought of as what usually called a SIEM plugin, collector, or connector in other platform. Example for Suricata IDS Eve JSON log is shown [here](https://github.com/defenxor/dsiem/blob/master/deployments/docker/conf/logstash/conf.d/70_siem-plugin-suricata.conf).*
    
* The output the above step is called *Normalized Event* because it represent logs from multiple different sources in a single format that has a set of common fields. This event is then sent to Dsiem through Logstash HTTP output plugin, and to Elasticsearch under index name pattern ```siem_events-*``` for further use.

* Dsiem correlates incoming normalized events based on the configured directive rules, perform threat intel and vulnerability lookups, and then generates an alarm if the rules conditions are met. This alarm is then written to a local log file, that is harvested by a local filebeat configured to send its content to Logstash.

* At the logstash end, there's another Dsiem [special config file](https://github.com/defenxor/dsiem/blob/master/deployments/docker/conf/logstash/conf.d/80_siem.conf) that reads those submitted alarms and push them to the final SIEM alarm index in Elasticsearch. This config file ensures that further updates made by Dsiem to the same alarm will also update the corresponding Elasticsearch document instead of creating a new one.
    
The end result of the above process is that now we can watch for new alarms and updates to an existing one just by monitoring a single Elasticsearch index.

## Installation

You can use Docker Compose or the release binaries to install Dsiem. Refer to the [Installation](/docs/Installation.md) doc for details.

## How to Contribute

Contributions are very welcome! Submit PR for bug fixes and additional tests, gist for logstash config files to parse device events, SIEM directive rules, or a new threat intel/vulnerability lookup plugins.

If you're not sure on what to do on a particular matter, feel free to open an <a href="https://github.com/defenxor/dsiem/issues"> issue</a> and discuss first.

## License

The project is licensed under <a href="https://github.com/defenxor/dsiem/blob/master/LICENSE">GPLv3</a>. Contributors are not required to sign any form of CAA/CLA or a like: We consider their acceptance of <a href="https://help.github.com/articles/github-terms-of-service/#6-contributions-under-repository-license">this Github terms of service clause</a> to be sufficient.