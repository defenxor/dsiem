# Dsiem Plugin

Dsiem plugin is a Logstash configuration file whose function is to clone events parsed by Logstash, convert them to a standard format called `Normalized Event`, and send them to Dsiem for processing.

## Normalized Event

The following table shows the fields of a `Normalized Event`:

| Field   |      Description      |  Mandatory | Usable in Correlation Rules |
|----------|-------------|------|------|
| timestamp | The original event timestamp in ISO8601 format, not to be confused with Logstash builtin `@timestamp`. | Yes | No, but this is used to detect out-of-order events and transit time.
| event_id | UUID for the event, typically generated using [Logstash UUID filter plugin](https://www.elastic.co/guide/en/logstash/current/plugins-filters-uuid.html) if the event doesn't already have one. | Yes | No
| title | Description of the event. | Yes | No
| sensor | String identifier of the device that produces/captures the event. Examples: hostname of the IPS device, firewall, or the processing Logstash node. | Yes | No
| src_ip | Source IP, should refer to the sender for network communication based events. For host-based events, use the host main IP address if it's available on the event's record, or just use `127.0.0.1`. | Yes | Yes
| dst_ip |    Destination IP, should refer to the receiver for network communication based events. Host-based events should just use the same address as `src_ip` or `127.0.0.1` |   Yes | Yes
| protocol |  Network protocol used, such as TCP, UDP, ICMP, etc. | No | Yes
| src_port | Source port number, typically refers to TCP or UDP ports, but may also be any identifying number like ICMP type number, etc. | No | Yes
| dst_port | Source port number, typically refers to TCP or UDP ports, but may also be any identifying number like ICMP type number, etc. |  No | Yes
| product | Product-type of the device that generates the event, i.e. firewall, IDS/IPS, etc. | Yes, if `plugin_id` or `plugin_sid` is empty | Yes, in [TaxonomyRule](./directive_and_alarm.md#about-directive-rules)
| category | The event's category, relative to the product type. For example, if the product type is firewall, event's category maybe `Allowed Traffic`,`Denied Traffic`, `Dropped Traffic`, `Port Scan` etc. | Yes, if `plugin_id` or `plugin_sid` is empty | Yes, in [TaxonomyRule](./directive_and_alarm.md#about-directive-rules)
| subcategory |  further breakdown of the event's category. For example, if the category is `Code Injection Attack`, subcategory maybe `SQL Injection`, `HTTP Parameter Injection`, etc. | No | Yes, in [TaxonomyRule](./directive_and_alarm.md#about-directive-rules)
| plugin_id | A unique number that identifies the plugin. For example, `1001` for Suricata eve.json based events as used in Dsiem default config (`1001` is also used in OSSIM by default for Suricata UnifiedThreat logs)  | Yes, if `product` or `category` is empty | Yes, in [PluginRule](./directive_and_alarm.md#about-directive-rules)
| plugin_sid |  A unique number that identifies the event *within* the plugin. |Yes, if `product` or `category` is empty | Yes, in [PluginRule](./directive_and_alarm.md#about-directive-rules)
| custom_label1 | A text identifier for an extra/custom field to use for correlation rules. | No | No
| custom_data1 |  The text content for the extra/custom field defined by `custom_label1`. | No | Yes
| custom_label2 | A text identifier for an extra/custom field to use for correlation rules. | No | No
| custom_data2 |  The text content for the extra/custom field defined by `custom_label2`. | No | Yes
| custom_label3 | A text identifier for an extra/custom field to use for correlation rules. | No | No
| custom_data3 |  The text content for the extra/custom field defined by `custom_label3`. | No | Yes

## Creating a Dsiem Plugin

Dsiem plugin can be created automatically from an existing index in Elasticsearch with the help of `dpluger` tool. The Logstash config file created by `dpluger` can then be used to filter and parse incoming events in order to produce normalized events and, with the help of [`80_siem.conf](https://github.com/defenxor/dsiem/blob/master/deployments/docker/conf/logstash/conf.d/80_siem.conf), send them to Dsiem for further processing.

There are two types of Dsiem plugin:
* SID-based plugin: produces normalized events to be processed later by a directive [`PluginRule`](./directive_and_alarm.md#about-directive-rules)
* Taxonomy-based plugin: produces normalized events for directive [`TaxonomyRule`](./directive_and_alarm.md#about-directive-rules).

Examples on how to create them with `dpluger` assistance are given below.

### Example 1: SID-based Plugin
Suppose your elasticsearch is located at http://elasticsearch:9200 and there is an index there named `suricata-*` for Suricata IDS that you want to create a plugin for. Here are the steps to do it:

* Download and extract the latest version of `dsiem-tools` from this project release page.

* Create an empty dpluger config file template to use:
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

### Example 2: SID-based Plugin with Generated Plugin SID

In the previous example, each type of event produced by Suricata has a uniq identifier in `alert.signature_id` that can be used for `plugin_sid` in the resulting normalized event.

But what if the source index doesn't have such field? in that case `dpluger` can be instructed to look for uniq entries in certain fields (such as title, etc.) and assign a `plugin_sid` number for each of them. To do that, just set `plugin_sid` to `collect:ES.field.name`, as in the example dpluger config file that we can use for McAfee NSP below:
```json
{
  "name": "mcafee-nsp",
  "type": "SID",
  "output_file": "70_siem-plugin-mcafee-nsp.conf",
  "index_pattern": "mcafee-nsp-*",
  "elasticsearch_address": "http://elasticsearch:9200",
  "identifier_field": "[fields][log_type]",
  "identifier_value": "mcafee-nsp",
  "identifier_filter": "",
  "field_mapping": {
    "title": "es:signature",
    "timestamp": "es:syslog_timestamp",
    "timestamp_format": "ISO8601",
    "sensor": "es:device_hostname",
    "plugin_id": "1832",
    "plugin_sid": "collect:signature",
    "product": "Intrusion Prevention System",
    "src_ip": "es:src_ip",
    "src_port": "es:src_port",
    "dst_ip": "es:dst_ip",
    "dst_port": "es:dst_port",
    "protocol": ""
  }
}
```
Using the above config, `dpluger` will gather uniq entries in `signature` field of `mcafee-nsp-*` index, and generate entries for `plugin_sid` field based that. The following shows an excerpt of the relevant part of the generated Logstash config file that does this:

```yaml
translate {
        field => "%[signature]"
        destination => "[plugin_sid]"
        dictionary => {
          "BlueCoat: Blue Coat BCAAA Stack Buffer Overflow Vulnerability" => "1"
          "P2P: TeamViewer Traffic Detected" => "2"
          "HTTP: Internet Media Tunneling through HTTP" => "3"
          "NETBIOS-SS: Windows SMB Remote Code Execution Vulnerability" => "4"
```
Running `dpluger` with `collect:` as shown above will also create the following TSV reference file:
```tsv
plugin	        id	sid	title
mcafee-nsp	1832	1	BlueCoat: Blue Coat BCAAA Stack Buffer Overflow Vulnerability
mcafee-nsp	1832	2	P2P: TeamViewer Traffic Detected
mcafee-nsp	1832	3	HTTP: Internet Media Tunneling through HTTP
mcafee-nsp	1832	4	NETBIOS-SS: Windows SMB Remote Code Execution Vulnerability
mcafee-nsp	1832	5	NETBIOS-SS: Illegal Secondary Transaction request seen
mcafee-nsp	1832	6	P2P: BitTorrent Meta-Info Retrieving
mcafee-nsp	1832	7	DoS: Cisco Syslog DoS
```
The content of that TSV file will be used as a starting point and lookup table on future `dpluger` runs. This means it is safe to run `dpluger` repeatedly against the same index (for instance, to update the lookup dictionary when new uniq values are added), because all of the previously detected titles will retain their SID number.

### Example 3: Taxonomy-based Plugin

Supply `-t` parameter to `dpluger` to generate a template for a taxonomy-based plugin:

```shell
$ ./dpluger create -i firewall-* -t Taxonomy
```

the resulting `dpluger_config.json` file would be:

```JSON
{
  "name": "suricata",
  "type": "Taxonomy",
  "output_file": "70_siem-plugin-suricata.conf",
  "index_pattern": "firewall-*",
  "elasticsearch_address": "http://elasticsearch:9200",
  "identifier_field": "INSERT_LOGSTASH_IDENTIFYING_FIELD_HERE (example: [application] or [fields][log_type] etc)",
  "identifier_value": "INSERT_IDENTIFYING_FIELD_VALUE_HERE (example: suricata)",
  "identifier_filter": "INSERT_ADDITIONAL_FILTER_HERE_HERE (example: and [alert])",
  "field_mapping": {
    "title": "es:INSERT_ES_FIELDNAME_HERE",
    "timestamp": "es:INSERT_ES_FIELDNAME_HERE",
    "timestamp_format": "INSERT_TIMESTAMP_FORMAT_HERE (example: ISO8601)",
    "sensor": "es:INSERT_ES_FIELDNAME_HERE",
    "product": "INSERT_PRODUCT_NAME_HERE",
    "category": "es:INSERT_ES_FIELDNAME_HERE",
    "subcategory": "es:INSERT_ES_FIELDNAME_HERE",
    "src_ip": "es:INSERT_ES_FIELDNAME_HERE",
    "src_port": "es:INSERT_ES_FIELDNAME_HERE",
    "dst_ip": "es:INSERT_ES_FIELDNAME_HERE",
    "dst_port": "es:INSERT_ES_FIELDNAME_HERE",
    "protocol": "es:INSERT_ES_FIELDNAME_HERE or INSERT_PROTOCOL_NAME_HERE"
  }
}
```
Notice how `plugin_id` and `plugin_sid` keys are replaced with `product`, `category`, and `subcategory`. From here on, the steps to complete the plugin is similar with those outlined in Example 1 above.

### Embed Custom Identifer block from file
For certain use-case, you can embed identifier block config directly from a file by specifying `identifier_block_source` in the dpluger config file. This way, `dpluger` will ignore the value of `identifier_field`, `identifier_value`, and `identifier_filter`, and import the content of the file directly under the first `filter` section.

For example, file `filter.conf` contain the following config:

```YAML
if [@metadata][log_type] == "sshd" and [sshd_result]
{
  mutate {
    id => "tag normalizedEvent 10008"
    add_field => {
      "[@metadata][siem_plugin_type]" => "sshd"
      "[@metadata][siem_data_type]" => "normalizedEvent"
    }
  }
}
```

To embed this directly to the resulting logstash config, you can set the `identifier_block_source` value to the path of `filter.conf`:

```JSON
...
"identifier_block_source": "filter.conf",
...
```

The content of `filter.conf` will fill the `filter` block inside the 1st and 2nd step of the resulting logstash config:

```YAML
filter {

# embedded from filter.conf

    if [@metadata][log_type] == "sshd" and [sshd_result] {
      mutate {
        id => "tag normalizedEvent 10008"
        add_field => {
          "[@metadata][siem_plugin_type]" => "sshd"
          "[@metadata][siem_data_type]" => "normalizedEvent"
        }
      }
    }

}
```