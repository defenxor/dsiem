# Dsiem 

[![CircleCI](https://circleci.com/gh/defenxor/dsiem.svg?style=shield&circle-token=def79b85071ad74a4bb86fd9d225bb09d00694c5)](https://circleci.com/gh/defenxor/dsiem) [![Go Report Card](https://goreportcard.com/badge/github.com/defenxor/dsiem)](https://goreportcard.com/report/github.com/defenxor/dsiem) [![Codecov](https://codecov.io/gh/defenxor/dsiem/branch/master/graph/badge.svg?token=3446slNekt)](https://codecov.io/gh/defenxor/dsiem) [![License: GPL v3](https://img.shields.io/badge/License-GPL%20v3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0) 

Dsiem is a security event correlation engine for [ELK stack](https://www.elastic.co/elk-stack), allowing the platform to be used as a dedicated and full-featured [SIEM](https://en.wikipedia.org/wiki/Security_information_and_event_management) system.

Dsiem provides [OSSIM](https://www.alienvault.com/products/ossim)-style correlation for normalized logs/events, perform lookup/query to threat intelligence and vulnerability information sources, and produces risk-adjusted alarms.

![Example Kibana Dashboard](/docs/images/kbn-dashboard.png)

## Features

* Runs in standalone or clustered mode with [NATS](https://nats.io/) as messaging bus between frontend and backend nodes. Along with ELK, this made the entire SIEM platform horizontally scalable.
* OSSIM-style correlation and directive rules, bridging easier transition from OSSIM.
* Alarms enrichment with data from threat intel and vulnerability information sources. Builtin support for [Moloch Wise](https://github.com/aol/moloch/wiki/WISE) (which supports Alienvault OTX and others) and Nessus CSV exports. Support for other sources can easily be implemented as [plugins](https://github.com/defenxor/dsiem/blob/master/docs/plugins.md#about-threat-intel-lookup-plugin).
* Instrumentation supported through Metricbeat and/or Elastic APM server. No need extra stack for this purpose.
* Builtin rate and back-pressure control, set the minimum and maximum events/second (EPS) received from Logstash depending on your hardware capacity and acceptable delays in event processing.
* Loosely coupled, designed to be composable with other infrastructure platform, and doesn't try to do everything. Loose coupling also means that it's possible to use Dsiem as an OSSIM-style correlation engine with non ELK stack if needed.
* Batteries included:
    * A directive conversion tool that reads OSSIM XML directive file and translate it to Dsiem JSON-style config.
    * A SIEM plugin creator tool that will read off an existing index pattern from Elasticsearch, and creates the necessary Logstash configuration to clone the relevant fields' content to Dsiem. The tool can also generate basic directive required by Dsiem to correlate received events and generate alarm.
    * A helper tool to serve Nessus CSV files over the network to Dsiem.
    * A light weight Angular web UI just for basic alarms management (closing, tagging), and easy pivoting to the relevant indices in Kibana to perform the actual analysis.
* Obviously a cloud-native, twelve-factor app, and all that jazz.

## How It Works

![Simple Architecture](https://github.com/defenxor/dsiem/blob/master/docs/images/simple-arch.png)

On the diagram above:

1. Log sources send their logs to Syslog/Filebeat, which then sends them to Logstash with a unique identifying field. Logstash then parses the logs using different filters based on the log sources type, and sends the results to Elasticsearch, typically creating a single index pattern for each log type (e.g. `suricata-*` for logs received from Suricata IDS, `ssh-*` for SSH logs, etc.).

1. Dsiem uses a special purpose logstash config file to clone incoming event from log sources, right after logstash has done parsing it. Through the same config file, the new cloned event is used (independently from the original event) to collect Dsiem required fields like Title, Source IP, Destination IP, and so on.
    
1. The output of the above step is called *Normalized Event* because it represent logs from multiple different sources in a single format that has a set of common fields. Those events are then sent to Dsiem through Logstash HTTP output plugin, and to Elasticsearch under index name pattern `siem_events-*`.

1. Dsiem correlates incoming normalized events based on the configured directive rules, perform threat intel and vulnerability lookups, and then generates an alarm if the rules conditions are met. The alarm is then written to a local log file, that is harvested by a local Filebeat configured to send its content to Logstash.

1. At the logstash end, there's another Dsiem [special config file](https://github.com/defenxor/dsiem/blob/master/deployments/docker/conf/logstash/conf.d/80_siem.conf) that reads those submitted alarms and push them to the final SIEM alarm index in Elasticsearch.
    
The final result of the above processes is that now we can watch for new alarms and updates to an existing one just by monitoring a single Elasticsearch index.

## Installation

You can use Docker Compose or the release binaries to install Dsiem. Refer to the [Installation Guide](https://github.com/defenxor/dsiem/blob/master/docs/installation.md) for details.

## Documentation

Currently available docs are located [here](https://github.com/defenxor/dsiem/blob/master/docs/).

## Reporting Bugs and Issues

Please submit bug and issue reports by opening a new Github [issue](https://github.com/defenxor/dsiem/issues/new). Security-sensitive information, like details of a potential security bug, may also be sent to devs@defenxor.com. The GPG public key for that address can be found [here](https://pgp.mit.edu/pks/lookup?search=devs%40defenxor.com).


## How to Contribute

Contributions are very welcome! Submit PR for bug fixes and additional tests, gist for Logstash config files to parse device events, SIEM directive rules, or a new threat intel/vulnerability lookup plugins.

If you're not sure on what to do on a particular matter, feel free to open an <a href="https://github.com/defenxor/dsiem/issues"> issue</a> and discuss first.

## License

The project is licensed under <a href="https://github.com/defenxor/dsiem/blob/master/LICENSE">GPLv3</a>. Contributors are not required to sign any form of CAA/CLA or a like: We consider their acceptance of <a href="https://help.github.com/articles/github-terms-of-service/#6-contributions-under-repository-license">this Github terms of service clause</a> to be sufficient.