# Threat Intelligence and Vulnerability Lookup Plugins

Threat intel plugin enriches content of an alarm whenever it involves a public IP address that is listed in one of the plugin backend databases. The same goes for Vulnerability lookup plugin, but here the search is done based on IP and port combination, and the alarm's IP address to lookup will also include any private IP addresses.

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

`CheckIP` will receive its `ip` parameter from SIEM alarm's source and destination IP addresses. The plugin should then check that address against its sources (e.g. by database lookups, API calls, etc.), and return `found=true` if there's a matching entry for it. If that's the case, Dsiem expects the plugin to also return more detail information in multiple `intel.Result` struct as follows:

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

Next clone this repository and test the build process for `dsiem` binary. Example on a Linux or OSX system:

```bash
$ git clone https://github.com/defenxor/dsiem
$ cd dsiem
$ go build ./cmd/dsiem
```

You should now have a `dsiem` binary in the current directory, and ready to start developing a plugin.

A quick way of creating a new intel plugin by using Wise as template is shown below. The same steps should also apply for making a new vulnerability lookup plugin based on Nesd.

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
$ ./dsiem serve | grep intel
{"level":"INFO","ts":"2018-11-20T21:35:04.238+0700","msg":"Adding intel plugin Myintel"}
{"level":"INFO","ts":"2018-11-20T21:35:04.239+0700","msg":"Adding intel plugin Wise"}
{"level":"INFO","ts":"2018-11-20T21:35:04.239+0700","msg":"Loaded 2 threat intelligence sources."}
```

And that's it. From here on you can start editing `contrib/intel/myintel/myintel.go` to implement your plugin's unique functionality. Don't forget to send PR when you're done ;).
