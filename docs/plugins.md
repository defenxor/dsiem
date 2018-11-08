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

TODO

## Developing a Vulnerability Lookup Plugin

TODO
