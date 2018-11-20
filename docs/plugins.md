# Plugins

There's 3 kinds of plugin in Dsiem: SIEM plugin, Threat Intel lookup plugin, and Vulnerability lookup plugin.

SIEM plugin is a logstash configuration file whose function is to clone events parsed by Logstash, normalise it to a standard format, and send it to Dsiem for processing. SIEM plugin can be created automatically from existing index in Elasticsearch with the help of  `dpluger` tool.

Threat intel plugin is used to enrich content of an alarm whenever it involves a public IP address that is listed in one of the plugin backend databases. The same goes for Vulnerability lookup plugin, but here the search is done based on IP and port combination, and the alarm IP address to lookup is not limited to just public IP addresses.

For now, threat intel and vulnerability lookup plugins can only be created by writing a Go package that implement the required interface.

## Creating a SIEM Plugin

* Download and extract the latest version of `dsiem-tools` from this project release page.

* Create an empty `dpluger` config file to use:

TODO

## Developing a Threat Intel Lookup Plugin

Intel lookup plugin is simply a Go package that implements the following interface:
```go
type Checker interface {
	CheckIP(ctx context.Context, ip string) (found bool, results []Result, err error)
	Initialize(config []byte) error
}
```

`Initialize` will receive its `config` content from the text defined in `configs/intel_*.json` file. This allows user to pass in
custom data in any format to the plugin to configure its behavior.

`CheckIP` will receive its `IP` parameter from SIEM alarm's source and destination IP addresses. The plugin should then check that address against its sources (e.g. by database lookups, API calls, etc.), and return `found=true` if there's a matching entry for that address. If that's the case, Dsiem expects the plugin to also return more detail information in multiple `intel.Result` struct as follows:

```go
// Result defines the struct that must be returned by an intel plugin
type Result struct {
	Provider string `json:"provider"`
	Term     string `json:"term"`
	Result   string `json:"result"`
}
```

You can see a working example of this in [Wise](https://github.com/defenxor/dsiem/blob/master/internal/pkg/plugin/wise/wise.go) intel plugin code. That plugin uses `Initialize` function to obtain Wise server address to use.



## Developing a Vulnerability Lookup Plugin

TODO
