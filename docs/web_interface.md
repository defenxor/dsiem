# Web Interface

Dsiem comes with an Angular-based web UI for managing alarms and quick pivoting to the relevant sections in Kibana for further analysis. In addition, there's also an example Kibana dashboard ready for import, and Elastic APM integration for instrumentation/tracing.

## Dsiem Web UI

TODO

## Kibana Dashboard

TODO

## APM Integration

***Note:** Dsiem currently uses a pre-release version of APM Go client library (v0.5.2). The release version (v1.0.0+) requires ELK stack v6.5 that we don't have plan to adopt yet.*

Dsiem comes integrated with Elastic APM for tracing purpose. Currently there are 5 custom transaction types created by Dsiem:

* Log Source to Frontend: measures the time it took for the event to reach Dsiem frontend. This is calculated as the `event's arrival time - the event's timestamp`.
* Frontend to Backend: measures the network transit time (including processing time by NATS ) between frontend and backend.
* Directive Event Processing: measures how long an event is processed by backend's backlog.
* Threat Intel Lookup: measures how long threat intel plugins process lookup request.
* Vulnerability Lookup: measures how long vulnerability lookup plugins process lookup request.

To enable APM integration, start Dsiem node with the following environment variables:

```shell
$ export ELASTIC_APM_SERVER_URL="http://[your-APM-server-address]:8200"
$ export ELASTIC_APM_SERVICE_NAME="dsiem"
$ export ELASTIC_APM_ACTIVE="true"
$ export DSIEM_APM="true"
$ ./dsiem serve
```
Example screenshot of Dsiem's APM dashboard:

![APM Dashboard for Dsiem](./images/apm.png)