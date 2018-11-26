# Web Interface

Dsiem comes with an Angular-based web UI for managing alarms and quick pivoting to the relevant sections in Kibana for further analysis. In addition, there's also an example Kibana dashboard ready for import, and Elastic APM integration for instrumentation/tracing.

## Dsiem Web UI

TODO

## Kibana Dashboard

TODO

## APM Integration

***Note:** APM integration is an experimental feature that is likely to change in the future. For instance, due to the ELK-stack version (6.4) in use, Dsiem currently cannot use APM's OpenTracing API which would enable a transaction to continue from frontend to backend node.*

Dsiem comes integrated with Elastic APM for tracing/instrumentation purpose. Currently there are 3 custom transactions created by Dsiem:

* Log Source to Frontend: measures the time it took for the event to reach Dsiem frontend. This is calculated as the `event's arrival time - the event's timestamp`.
* Frontend to Backend: measures the event's transit time in the message queue.
* Directive Event Processing: measures how long an event is processed by backend's backlog.


To enable this functionality, start Dsiem node with the following environment variables:

```shell
$ export ELASTIC_APM_SERVER_URL="http://[your-APM-server-address]:8200"
$ export ELASTIC_APM_SERVICE_NAME="dsiem"
$ export ELASTIC_APM_ACTIVE="true"
$ export DSIEM_APM="true"
$ ./dsiem serve
```
Example of D