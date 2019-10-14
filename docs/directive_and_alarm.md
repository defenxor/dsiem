# Directive and Alarm

Similar to [OSSIM](https://www.alienvault.com/products/ossim), Dsiem directive contains a set of rules that will be used to evaluate incoming [Normalized Events](./dsiem_plugin.md#normalized-event). Directive triggers alarms when enough of its rules condition are met.

## About Directive Rules

There are two types of directive rules in Dsiem, `PluginRule` and `TaxonomyRule`. As suggested in the [Normalized Events](./dsiem_plugin.md#normalized-event) table, `PluginRule` differentiates events based on `PluginID` and `Plugin_SID` fields, while TaxonomyRule uses `Product`, `Category`, and optionally `Subcategory` fields.

`PluginRule` should be used if you want to do correlation based on specific events produced by specific brand of devices. On the other hand, `TaxonomyRule` allows correlation to be done based on a group of events that share similar characteristic.

As an example, suppose you have the following security devices in your network: Suricata IDS, SomebrandNG IDS, Pfsense Firewall, SomeRouterNG Firewall. `PluginRule` will allow you to define a directive that says, "Raise alarm if Suricata IDS detects SQL injection or XSS attacks that isn't blocked by SomeRouterNG Firewall". In contrast, `TaxonomyRule` allow definition of a more general directive that says, "Raise alarm if an IDS detects web application attack that isn't blocked by firewall".

Despite its obvious flexibility, `TaxonomyRule` does require you to maintain a custom classification/taxonomy scheme that isn't required by `PluginRule`. In the above example, using `TaxonomyRule` means you will have to know and classify which events from Suricata or SomebrandNG are "web application attack", and which events in PfSense and SomeRouterNG means "not blocking".

As a general guide, `TaxonomyRule` should be preferred when there are similar types of product (like IDS and firewall above) to cover since it will prevent a lot of redundant directives. But if that's not the case, `PluginRule` will likely offer an easier to maintain plugin/parser configuration.

## Directive and Rules Processing

The following is an example of a dsiem directive that has several `PluginRule`.

```json
{
  "name": "Ping Flood from SRC_IP",
  "kingdom": "Reconnaissance & Probing",
  "category": "Misc Activity",
  "id": 1,
  "priority": 3,
  "rules" : [
    { "name": "ICMP Ping", "type": "PluginRule", "stage": 1, "plugin_id": 1001, 
      "plugin_sid": [ 2100384 ], "occurrence": 1, "from": "HOME_NET", "to": "ANY",
      "port_from": "ANY", "port_to": "ANY", "protocol": "ICMP", 
      "reliability": 1, "timeout": 0 
    },
    { "name": "ICMP Ping", "type": "PluginRule", "stage": 2, "plugin_id": 1001,
      "plugin_sid": [ 2100384 ], "occurrence": 5, "from": ":1", "to": "ANY",
      "port_from": "ANY", "port_to": "ANY", "protocol": "ICMP",
      "reliability": 5, "timeout": 600 
    },
    { "name": "ICMP Ping", "type": "PluginRule", "stage": 3, "plugin_id": 1001,
      "plugin_sid": [ 2100384 ], "occurrence": 500, "from": ":1", "to": "ANY", 
      "port_from": "ANY", "port_to": "ANY", "protocol": "ICMP", 
      "reliability": 10, "timeout": 3600
    }
  ]
}
```
Notice that the directive `Ping Flood from SRC_IP` above has 3 `rules`, each having a `stage` from 1 to 3 and different value for `reliability` and `timeout` keys.

When an incoming event matches a directive's stage 1 rule condition, dsiem will create a backlog object associated with that directive to track the progress of this potential alarm.

Dsiem will then wait for incoming events that match the next rule (i.e. stage 2) rule condition, until one of these conditions happen:
- the count for matching events reach the stage `occurrence` value, in which case evaluation will continue to next rule (i.e. stage 3); or
- `timeout` duration (in seconds) has elapsed since the first matched event, in which case the backlog will be discarded.

The above repeats until evaluation of the last rule completes — after which the backlog object will finally be removed.

## Triggering Alarm

Dsiem calculates a risk value for backlog when the current stage's rule condition has been matched by incoming events. An alarm will be triggered *when the backlog risk value is ≥ 1*. Risk value calculation is based on this formula:

```go
Risk = (reliability * priority * asset value)/25
```

Where: 
- Reliability (1 to 10): the active rule `reliability` value. Higher reliability means higher confidence on the likelihood that the (potential) alarm is a true positive.
- Priority (1 to 5): The directive's `priority` value. This reflects the directive level of importance. Those with more severe impact should be given a higher priority.
- Asset Value (1 to 5): From the asset value set on `assets_*.json` config files. This reflects the criticality of the target/impacted assets. Higher value should be assigned to more critical assets. 

Given the above, we can calculate that risk value of any backlog will vary from 0.4 when all parameters are at their minimum, to 10 when all parameters are at their maximum. Since alarm will only be created when that value is ≥ 1, we can also deduce that the risk range of any alarm will be from 1 to 10.

By using [Dsiem startup parameter](commands.md#dsiem-command-flags), you can then configure when should an alarm be labeled as Low, Medium or High risk based on that range. The default thresholds are:
- Low risk ⟶ risk value of 1 to less than 3
- Medium risk ⟶ risk value of 3 to 6
- High risk ⟶ risk value of more than 6 to 10

*The startup parameters `medRiskMin` and `medRiskMax` control number `3` and `6` above respectively.*

### Example Directive ⟶ Alarm Processing

Let's say Dsiem is configured with the above `Ping Flood from SRC_IP` directive, and an `assets_*.json` that defines an asset value of 4 for `10.0.0.0/8` network.

When Dsiem receives the following series of events:
1. plugin_id: 1001, plugin_sid: 2100384, protocol: ICMP, src_ip: 10.0.0.1, dst_ip: 10.0.0.2
1. plugin_id: 1001, plugin_sid: 2100384, protocol: ICMP, src_ip: 10.0.0.1, dst_ip: 10.0.0.3
1. plugin_id: 1001, plugin_sid: 2100384, protocol: ICMP, src_ip: 10.0.0.2, dst_ip: 10.0.0.1
1. plugin_id: 1001, plugin_sid: 2100384, protocol: ICMP, src_ip: 10.0.0.1, dst_ip: 10.0.0.4
1. plugin_id: 1001, plugin_sid: 2100384, protocol: ICMP, src_ip: 10.0.0.1, dst_ip: 10.0.0.5 (repeated 13x)

The following processing will take place:

* On 1st event:
  * the event matches directive 1st rule condition because:
    * they have the same value for `plugin_id`, `plugin_sid`, and `protocol`.
    * the event `src_ip` (10.0.0.1) matches the directive `from` condition of `HOME_NET`. This is because `HOME_NET` refers to any addresses defined in `assets_*.json` file.
    * the event `dst_ip` (10.0.0.2) matches the directive `to` condition of `ANY`. This is because `ANY` will match all possible value.
  * backlog #1 is created to track this potential alarm.
  * backlog #1 has fulfilled all of its condition, including its required number of `occurrence` (1 event). This starts a risk calculation of `risk = (reliability * priority * asset value)/25 = (1 * 3 * 4)/25 = 0.48`, so no alarm will be created yet.
  * backlog #1 now moves to the 2nd correlation stage and starts monitoring for incoming events that would trigger the 2nd stage rule condition.
* On 2nd event:  
  * the event matches backlog #1 2nd rule condition because the event `src_ip` matches the directive `from` value of `:1`. That `:1` denotes a reference to the `from` value matched in the first rule, which in this case is `10.0.0.1`.
* On 3rd event:
  * the event doesn't match backlog #1 2nd rule condition because of the mismatch in `from` condition and the event's `src_ip`.
  * since this event does match the directive 1st rule, a new backlog #2 will be created to track it separately from backlog #1.
* On 4th event:
  * the event matches backlog #1 2nd rule condition. This will raise its event count from 1 to 2 for the 2nd stage.
  * the event doesnt match backlog #2 2nd rule condition.
* On 5th to 17th event:
  * 5th to 7th events match backlog #1 2nd rule condition and cause its event count to reach the `occurrence` limit (5 events), so this will recalculate the risk value to be `risk = (5 * 3 * 2)/25 = 2.4`. **This will generate a Low risk alarm**, and will also move backlog #1 to the 3rd correlation stage.  
  * the next 10 events (8-17th) match backlog #1 3rd rule condition, and will again recalculate the risk value to be `risk = (10 * 3 * 4)/25 = 4.8`. This results in an update of the associated alarm's risk label from Low to **Medium**.

  Backlog #1 now has no more rules to process and will immediately be deleted from memory. As for backlog #2, it will keep waiting for event matching its 2nd rule for 10 minutes (600s), after which it will expire and also be deleted.

## Creating a Dsiem Directive

Basic Dsiem directives can be created automatically by parsing TSV files produced by `dpluger` tool during [Dsiem plugin creation process](dsiem_plugin.md#example-2-sid-based-plugin-with-generated-plugin-sid). These directives are basic because they're only looking for events that have identical `PluginID` and `PluginSID` combination, and identical source and destination IP address pair.

For example, given a `sangfor_plugin-sids.tsv` file with the following content:
```tsv
plugin      id      sid     title
sangforIPS  20001   1       Botnet
sangforIPS  20001   2       Abnormal Connection
```

we can use the following `dpluger` command:
```console
./dpluger directive -i 3001 -f sangfor_plugin-sids.tsv
```
To automatically generate this `directives_dsiem.json` file:
```json
{
  "directives": [
    {
      "id": 3001,
      "name": "Botnet (SRC_IP to DST_IP)",
      "priority": 3,
      "kingdom": "Environmental Awareness",
      "category": "Misc Activity",
      "rules": [
        {
          "name": "Botnet",
          "stage": 1,
          "plugin_id": 20001,
          "plugin_sid": [
            1
          ],
          "occurrence": 1,
          "from": "ANY",
          "to": "ANY",
          "type": "PluginRule",
          "port_from": "ANY",
          "port_to": "ANY",
          "protocol": "TCP/IP",
          "reliability": 1,
          "timeout": 0
        },
        {
          "name": "Botnet",
          "stage": 2,
          "plugin_id": 20001,
          "plugin_sid": [
            1
          ],
          "occurrence": 10,
          "from": ":1",
          "to": ":1",
          "type": "PluginRule",
          "port_from": "ANY",
          "port_to": "ANY",
          "protocol": "TCP/IP",
          "reliability": 5,
          "timeout": 3600
        },
        {
          "name": "Botnet",
          "stage": 3,
          "plugin_id": 20001,
          "plugin_sid": [
            1
          ],
          "occurrence": 10000,
          "from": ":1",
          "to": ":1",
          "type": "PluginRule",
          "port_from": "ANY",
          "port_to": "ANY",
          "protocol": "TCP/IP",
          "reliability": 10,
          "timeout": 21600
        }
      ]
    },
    {
      "id": 3002,
      "name": "Abnormal Connection (SRC_IP to DST_IP)",
      "priority": 3,
      "kingdom": "Environmental Awareness",
      "category": "Misc Activity",
      "rules": [
        {
          "name": "Abnormal Connection",
          "stage": 1,
          "plugin_id": 20001,
          "plugin_sid": [
            2
          ],
          "occurrence": 1,
          "from": "ANY",
          "to": "ANY",
          "type": "PluginRule",
          "port_from": "ANY",
          "port_to": "ANY",
          "protocol": "TCP/IP",
          "reliability": 1,
          "timeout": 0
        },
        {
          "name": "Abnormal Connection",
          "stage": 2,
          "plugin_id": 20001,
          "plugin_sid": [
            2
          ],
          "occurrence": 10,
          "from": ":1",
          "to": ":1",
          "type": "PluginRule",
          "port_from": "ANY",
          "port_to": "ANY",
          "protocol": "TCP/IP",
          "reliability": 5,
          "timeout": 3600
        },
        {
          "name": "Abnormal Connection",
          "stage": 3,
          "plugin_id": 20001,
          "plugin_sid": [
            2
          ],
          "occurrence": 10000,
          "from": ":1",
          "to": ":1",
          "type": "PluginRule",
          "port_from": "ANY",
          "port_to": "ANY",
          "protocol": "TCP/IP",
          "reliability": 10,
          "timeout": 21600
        }
      ]
    }
  ]
}
```
The generated directives above have the following characteristics:
- Directive IDs are assigned sequentially starting with the number provided by the `-i` parameter to `dpluger`. This parameter is required to prevent conflicting IDs with directives already defined in other files.
- Each directive has 3 correlation rule stages:
  - stage 1: match a single event that has similar `PluginID` and `PluginSID` combination as specified by each row of the TSV file.
  - stage 2: has similar condition with stage 1, with an added requirement for the events to also match stage 1 source and destination IP addresses. This stage is set to match up to 10 events within 3,600 seconds (1 hour).
  - stage 3: has similar condition with stage 2, but is setup to match up to 10,000 events within 6 hour.
- The directive priority value (1) and the correlation rules reliability value (for each consecutive stages: 1, 5, 10) are set so that the directive will trigger a low risk alarm when stage 2 receives its last matching event. This is true if the events source or destination IP address include an asset whose value are at least 2 (the default asset value), as that will cause the last event for the 2nd stage to produce a risk value of 1.2.
