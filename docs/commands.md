# Dsiem Command and Tools

Dsiem comes with the following executable files:

* **dsiem** : the main dsiem executable.
* **dpluger** : for creating [SIEM plugin](./plugins.md) automatically by reading the fields of existing Elasticsearch index.
* **dtester** : for sending fake events that match your directive rules in order to test them. Dtester can send directly to Dsiem (therefore emulating Logstash), or save the events to a log file to be harvested by Filebeat for testing end-to-end processing pipelines.
* **nesd** : a [vulnerability lookup plugin](./plugins.md) that serves Nessus CSV export files over the network to be queried by Dsiem.
* **ossimcnv** : for converting OSSIM directive XML file (e.g. userdirectives.xml) to Dsiem's directive file format (JSON).

The main `dsiem` command is distributed under `dsiem-server-linux-amd64.zip` file on the Release page. The rest of the commands are inside `dsiem-tools-${os}-amd64.zip` file, where `os` is one of linux, darwin, or windows.

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
  help        Help about any command
  run         Creates logstash plugin for dsiem
  version     Print the version and build information

Flags:
  -c, --config string   config file to use (default "dpluger_config.json")
  -h, --help            help for dpluger

Use "dpluger [command] --help" for more information about a command.
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

## Example Usage

* To start dsiem in standalone mode:
```shell
$ ./dsiem serve
```
* To start dsiem with a maximum processing rate of 5,000 events/sec:
```shell
$ ./dsiem serve -e 5000
```
