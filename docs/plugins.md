# Plugins

There's 3 kinds of plugin in Dsiem: SIEM plugin, Threat Intel lookup plugin, and Vulnerability lookup plugin.

SIEM plugin is a Logstash configuration file whose function is to clone events parsed by Logstash, normalizes it to a standard format, and send it to Dsiem for processing. SIEM plugin can be created automatically from existing index in Elasticsearch with the help of  `dpluger` tool.

Threat intel plugin enriches content of an alarm whenever it involves a public IP address that is listed in one of the plugin backend databases. The same goes for Vulnerability lookup plugin, but here the search is done based on IP and port combination, and the alarm IP address to lookup will also be done against private IP addresses.

For now, threat intel and vulnerability lookup plugins can only be created by writing a Go package that implement the required interface.

## Creating a SIEM Plugin

Suppose your elasticsearch is located at http://elasticsearch:9200 and there is an index there named `suricata-*` for Suricata IDS that you want to create a plugin for. Here are the steps to do it:

* Download and extract the latest version of `dsiem-tools` from this project release page.

* Create an empty `dpluger` config file template to use:
  ```shell
  $ ./dpluger create -a http://elasticsearch:9200 -i "suricata-*" -n "suricata" -c dpluger_suricata.json
  ```
* The above will create a dpluger config file named `dpluger_suricata.json` in the current directory. The content of the file will be something like this:
```json
{
  "name": "suricata",
  "type": "SID",
  "output_file": "70_siem-plugin-suricata.conf",
  "index_pattern": "suricata-*",
  "elasticsearch_address": "http://elasticsearch:9200",
  "identifier_field": "INSERT_LOGSTASH_IDENTIFYING_FIELD_HERE (example: [application] or [fields][log_type] etc)",
  "identifier_value": "INSERT_IDENTIFYING_FIELD_VALUE_HERE (example: suricata)",
  "identifier_filter": "INSERT_ADDITIONAL_FILTER_HERE_HERE (example: and [alert])",
  "field_mapping": {
    "title": "es:INSERT_ES_FIELDNAME_HERE",
    "timestamp": "es:INSERT_ES_FIELDNAME_HERE",
    "timestamp_format": "INSERT_TIMESTAMP_FORMAT_HERE (example: ISO8601)",
    "sensor": "es:INSERT_ES_FIELDNAME_HERE",
    "plugin_id": "INSERT_PLUGIN_NUMBER_HERE",
    "plugin_sid": "es:INSERT_ES_FIELDNAME_HERE or collect:INSERT_ES_FIELDNAME_HERE",
    "product": "INSERT_PRODUCT_NAME_HERE",
    "src_ip": "es:INSERT_ES_FIELDNAME_HERE",
    "src_port": "es:INSERT_ES_FIELDNAME_HERE",
    "dst_ip": "es:INSERT_ES_FIELDNAME_HERE",
    "dst_port": "es:INSERT_ES_FIELDNAME_HERE",
    "protocol": "es:INSERT_ES_FIELDNAME_HERE or INSERT_PROTOCOL_NAME_HERE"
  }
}
```
* The next step is to edit that file so the field references and identifiers match with the actual field names in the target Elasticsearch `suricata-*` index. For index generated from Suricata Eve JSON format, which is also used in the [example Docker Compose deployments](https://github.com/defenxor/dsiem/tree/master/deployments/docker), the final config should be something like this:

```json
{
  "name": "suricata",
  "type": "SID",
  "output_file": "70_siem-plugin-suricata.conf",
  "index_pattern": "suricata-*",
  "elasticsearch_address": "http://elasticsearch:9200",
  "identifier_field": "[application]",
  "identifier_value": "suricata",
  "identifier_filter": "and [alert]",
  "field_mapping": {
    "title": "es:alert.signature",
    "timestamp": "es:timestamp",
    "timestamp_format": "ISO8601",
    "sensor": "es:host.name",
    "plugin_id": "1001",
    "plugin_sid": "es:alert.signature_id",
    "product": "Intrusion Detection System",
    "category": "es:alert.category",
    "src_ip": "es:src_ip",
    "src_port": "es:src_port",
    "dst_ip": "es:dest_ip",
    "dst_port": "es:dest_port",
    "protocol": "es:proto"
  }
}
```
* After that we can start `dpluger` again with `run` command. This will verify the existence of each field on the target Elasticsearch index, and then create a ready to use Logstash configuration file.

```bash
$ ./dpluger run -c dpluger_suricata.json
Creating plugin (logstash config) for suricata, using ES: http://elasticsearch:9200 and index pattern: suricata-*
2018-11-24T22:52:32.686+0700    INFO    Found ES version 6.4.2
Checking existence of field alert.signature... OK
Checking existence of field timestamp... OK
Checking existence of field host.name... OK
Checking existence of field alert.signature_id... OK
Checking existence of field alert.category... OK
Checking existence of field src_ip... OK
Checking existence of field src_port... OK
Checking existence of field dest_ip... OK
Checking existence of field dest_port... OK
Checking existence of field proto... OK
Logstash conf file created.
```
* The generated Logstash config file (i.e. a Dsiem SIEM plugin) will be  [`70_siem-plugin-suricata.conf`](https://github.com/defenxor/dsiem/blob/master/deployments/docker/conf/logstash/conf.d/70_siem-plugin-suricata.conf) located in the current directory.
To use the plugin, just copy it to Logstash configuration directory and reload Logstash.


## About Threat Intel Lookup Plugin

Intel lookup plugin is simply a Go package that implements the following interface:
```go
type Checker interface {
	CheckIP(ctx context.Context, ip string) (found bool, results []Result, err error)
	Initialize(config []byte) error
}
```

`Initialize` will receive its `config` content from the text defined in `configs/intel_*.json` file. This allows user to pass in
custom data in any format to the plugin to configure its behavior.

`CheckIP` will receive its `ip` parameter from SIEM alarm's source and destination IP addresses. The plugin should then check that address against its sources (e.g. by database lookups, API calls, etc.), and return `found=true` if there's a matching entry for that address. If that's the case, Dsiem expects the plugin to also return more detail information in multiple `intel.Result` struct as follows:

```go
// Result defines the struct that must be returned by an intel plugin
type Result struct {
	Provider string `json:"provider"`
	Term     string `json:"term"`
	Result   string `json:"result"`
}
```

You can see a working example of this in [Wise](https://github.com/defenxor/dsiem/blob/master/internal/pkg/plugin/wise/wise.go) intel plugin code. The plugin uses `Initialize` function to obtain Wise URL to use from the JSON [config file](https://github.com/defenxor/dsiem/blob/master/configs/intel_wise.json).

```JSON
{
  "intel_sources": [
    {
      "name": "Wise",
      "plugin": "Wise",
      "type": "IP",
      "enabled": true,
      "config": "{ \"url\" : \"http://wise:8081/ip/${ip}\" }"
    }
  ]
}
```

## About Vulnerability Lookup Plugin

Vulnerability lookup plugin is a Go package that implements the following interface:

```go
type Checker interface {
	CheckIPPort(ctx context.Context, ip string, port int) (found bool, results []Result, err error)
	Initialize(config []byte) error
}
```

The difference with intel plugin is that `CheckIPPort` here will receive `ip` and `port` combination instead of just `ip`. Those parameters will also come from alarm data, like source IP and source port, or destination IP and destination port.

A working example of this can be found in [Nesd](https://github.com/defenxor/dsiem/blob/master/internal/pkg/plugin/nesd/nesd.go) plugin code. The plugin uses `Initialize` function to obtain Nesd URL to use from the JSON [config file](https://github.com/defenxor/dsiem/blob/master/configs/vuln_nessus.json).

## Developing Intel or Vulnerability Lookup Plugin

First you need a working Go development environment. Just follow the instruction from [here](https://golang.org/doc/install) to get started.

Next clone this repository and test the build process for `dsiem` binary. Example on Linux or OSX system would be:

```bash
$ git clone https://github.com/defenxor/dsiem
$ cd dsiem
$ go build ./cmd/dsiem
```

You should now have a `dsiem` binary in the current directory, and ready to start developing a plugin.

A quick way of creating a new intel plugin by copying Wise is shown below. The same steps should also apply for making a new vulnerability lookup plugin based on Nesd.

```bash
# prepare the new plugin files based on wise
$ mkdir -p contrib/intel/myintel 
$ cp internal/pkg/plugin/wise/wise.go contrib/intel/myintel/myintel.go

# replace wise -> myintel and Wise -> Myintel in the code
$ sed -i 's/wise/myintel/g; s/Wise/Myintel/g' contrib/intel/myintel/myintel.go

# do the same for config file
$ cp configs/intel_wise.json configs/intel_myintel.json
$ sed -i 's/Wise/Myintel/g; s/wise/myintel/g' configs/intel_myintel.json

# insert entry in xcorrelator and make sure it's formatted correctly
$ sed -i 's/^)/_ \"github.com\/defenxor\/dsiem\/contrib\/intel\/myintel\"\)/g' internal/pkg/dsiem/xcorrelator/plugins.go
$ gofmt -s -w internal/pkg/dsiem/xcorrelator/plugins.go

# rebuild dsiem binary to include the new plugin
$ go build ./cmd/dsiem
```

After that, you can start dsiem and verify that the plugin is loaded correctly like so:

```bash
./dsiem serve | grep intel
{"level":"INFO","ts":"2018-11-20T21:35:04.238+0700","msg":"Adding intel plugin Myintel"}
{"level":"INFO","ts":"2018-11-20T21:35:04.239+0700","msg":"Adding intel plugin Wise"}
{"level":"INFO","ts":"2018-11-20T21:35:04.239+0700","msg":"Loaded 2 threat intelligence sources."}
```

And that's it. From here on you can start editing `contrib/intel/myintel/myintel.go` to implement your plugin's unique functionality. Don't forget to send PR when you're done ;).
