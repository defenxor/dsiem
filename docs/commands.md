# Dsiem Command and Tools

Dsiem comes with the following executable files:

* **dsiem** : the main dsiem executable for running the correlation engine.
* **dpluger** : for creating [Dsiem plugin](./dsiem_plugin.md) automatically by reading the fields of existing Elasticsearch index. It can also create a basic [Dsiem directive](./directive_and_alarm.md) that will trigger alarm when there's multiple occurrence of the same event.
* **dtester** : for sending simulated events that match your directive rules in order to test them. Dtester can send directly to Dsiem (therefore emulating Logstash), or save the events to a log file to be harvested by Filebeat for testing end-to-end processing pipelines.
* **nesd** : a [vulnerability lookup plugin](./ti_vuln_plugins.md) that serves Nessus CSV export files over the network to be queried by Dsiem.
* **ossimcnv** : for converting OSSIM directive XML file (e.g. userdirectives.xml) to Dsiem's directive file format (JSON).

On the [Release page](https://github.com/defenxor/dsiem/releases), the main `dsiem` command is distributed in `dsiem-server_${os}_amd64.zip` and the others in `dsiem-tools_${os}-amd64.zip`, where `os` is one of linux, darwin, or windows.

## Usage Information

All executables and their sub-commands have `-h` or `--help` flag that will outline and describe all available parameters. For example:

```shell
$ ./dpluger -h

Dpluger reads existing elasticsearch index pattern and creates a Dsiem logstash
config file (i.e. a plugin) from it.

Usage:
  dpluger [command]

Available Commands:
  create      Creates an empty config template for dpluger
  directive   Creates a DSIEM directive file from dpluger TSV
  help        Help about any command
  run         Creates logstash plugin for dsiem
  version     Print the version and build information

Flags:
  -c, --config string   config file to use (default "dpluger_config.json")
  -h, --help            help for dpluger

Use "dpluger [command] --help" for more information about a command
```

```shell
$ ./dpluger run -h
Creates logstash plugin for dsiem

Usage:
  dpluger run [flags]

Flags:
  -h, --help       help for run
  -v, --validate   Check whether each refered ES field exists on the target index (default true)

Global Flags:
  -c, --config string   config file to use (default "dpluger_config.json")
```

Each flag can be configured through command line parameter or environment variable. As an example, it is possible to execute `./dpluger run` above with `validate` flag set to `false` like this:

```shell
$ ./dpluger run --validate=false
```
or
```shell
$ export DSIEM_VALIDATE=false
$ ./dpluger run
```

Notice how the environment variable above starts with `DSIEM_` string. The same applies for all parameters, so to configure the `config` flag, you will need to set environment variable `DSIEM_CONFIG`, and so on.

Another example on this can be seen in the <a href="https://github.com/defenxor/dsiem/blob/master/deployments/docker/docker-compose-cluster.yml">docker compose file for cluster mode</a>, which uses this behaviour to assign two `dsiem` containers as either frontend or backend.

## Dsiem Command Flags

Dsiem main command has many startup flags to customize its runtime behavior. For instance, you can use custom tags and status for alarms, or change the risk calculation threshold to better reflect your team security analysis process.

The available flags is shown below.

```shell
./dsiem serve --help

Start dsiem server in a standalone or clustered deployment mode (either as frontend or backend).

Frontends listen for normalized events from logstash and distribute them to backends through NATS message queue.
Frontends also serve incoming request for configuration management from web UI.

Backends receive events on the message queue channel, perform correlation based on configured directive rules, 
and then send results/alarms to elasticsearch through local filebeat.

Standalone mode perform both frontend and backend functions in a single dsiem instance directly, without the need for
external message queue.

Usage:
  dsiem serve [flags]

Flags:
  -a, --address string         IP address for the HTTP server to listen on (default "0.0.0.0")
      --apm                    Enable elastic APM instrumentation
  -c, --cacheDuration int      Cache expiration time in minutes for intel and vuln query results (default 10)
      --frontend string        Frontend URL to pull configuration from, e.g. http://frontend:8080 (used only by backends).
  -h, --help                   help for serve
  -n, --holdDuration int       Duration in seconds before resetting overload condition state (default 10)
  -d, --maxDelay int           Max. processing delay in seconds before throttling incoming events (default 180)
  -e, --maxEPS int             Max. number of incoming events/second (default 1000)
      --medRiskMax int         Maximum alarm risk value to be classified as Medium risk. Higher value than this will be classified as High risk (default 6)
      --medRiskMin int         Minimum alarm risk value to be classified as Medium risk. Lower value than this will be classified as Low risk (default 3)
  -l, --minAlarmLifetime int   Min. alarm lifetime in minutes. Backlog won't expire sooner than this regardless rule timeouts. This is to support processing of delayed events
  -i, --minEPS int             Min. events/second rate allowed when throttling incoming events (default 100)
  -m, --mode string            Deployment mode, can be set to standalone, cluster-frontend, or cluster-backend (default "standalone")
      --msq string             Nats address to use for frontend - backend communication. (default "nats://dsiem-nats:4222")
      --node string            Unique node name to use when deployed in cluster mode.
  -p, --port int               TCP port for the HTTP server to listen on (default 8080)
      --pprof                  Enable go pprof on the web interface
  -s, --status strings         Alarm status to use, the first one will be assigned to new alarms (default [Open,In-Progress,Closed])
  -t, --tags strings           Alarm tags to use, the first one will be assigned to new alarms (default [Identified Threat,False Positive,Valid Threat,Security Incident])
      --trace                  Generate 10 seconds trace file for debugging.
      --websocket              Enable websocket endpoint that streams events/second measurement data
      --writeableConfig        Whether to allow configuration file update through HTTP

Global Flags:
      --debug   Enable debug messages for tracing and troubleshooting
      --dev     Enable development environment specific setting
```

We try to provide sensible defaults, so there's no need to supply additional flags unless there's a specific need to override them. For instance, in order to run dsiem in standalone mode you only need to run:
```shell
$ ./dsiem serve
```
Or if you want to also cap the maximum processing rate to 5,000 events/sec:
```shell
$ ./dsiem serve -e 5000
```

You can also see more examples of dsiem startup parameters in the Docker Compose files referenced on the [Installation](./installation.md) doc.