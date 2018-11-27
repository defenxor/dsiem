# Directive and Alarm

Similar to [OSSIM](https://www.alienvault.com/products/ossim), Dsiem directive contains a set of rules that will be used to evaluate incoming [Normalized Events](./dsiem_plugin.md#normalized-event). Directive triggers alarms when enough of its rules condition are met.

The following is an example of a dsiem directive.

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
      "reliability": 6, "timeout": 600 
    },
    { "name": "ICMP Ping", "type": "PluginRule", "stage": 3, "plugin_id": 1001,
      "plugin_sid": [ 2100384 ], "occurrence": 500, "from": ":1", "to": "ANY", 
      "port_from": "ANY", "port_to": "ANY", "protocol": "ICMP", 
      "reliability": 8, "timeout": 3600
    },
    { "name": "ICMP Ping", "type": "PluginRule", "stage": 4, "plugin_id": 1001,
      "plugin_sid": [ 2100384 ], "occurrence": 10000, "from": ":1", "to": "ANY", 
      "port_from": "ANY", "port_to": "ANY", "protocol": "ICMP", 
      "reliability": 10, "timeout": 3600
    }
  ]
}
```
Notice that the directive `Ping Flood from SRC_IP` above has 4 `rules`, each having a `stage` from 1 to 4 and different value for `reliability` and `timeout` keys.

When an incoming event match a directive's stage 1 rule condition, dsiem will create a backlog object associated with that directive to track the progress of this potential alarm.

Dsiem will then wait for incoming events that match the next rule (i.e. stage 2) rule condition, until one of these conditions happen:
- the count for matching events reach the stage `occurrence` value, in which case evaluation will continue to next rule (i.e. stage 3); or
- `timeout` duration (in seconds) has elapsed since the last matched event, in which case the backlog will be discarded.

The above repeats until evaluation of the last rule completes — after which the backlog object will finally be removed.

## Triggering Alarm

Dsiem calculates a risk value for backlog whenever that backlog moves to the next rule. An alarm will be triggered *when the backlog risk value is ≥ 1*. Risk value calculation is based on this formula:

```go
Risk = (reliability * priority * asset value)/25
```

Where: 
- Reliability (1 to 10): the active rule `reliability` value. Higher reliability means higher confidence on the likelihood that the (potential) alarm is a true positive.
- Priority (1 to 5): The directive's `priority` value. This reflects the directive level of importance. Those with more severe impact should be given a higher priority.
- Asset Value (1 to 5): From the asset value set on `assets_*.json` config files. This reflects the criticality of the target/impacted assets. Higher value should be assigned to more critical assets. 

Given the above, we can calculate that risk value of any backlog will vary from 0.4 when all parameters are at their minimum, to 10 when all parameters are at their maximum. Since alarm will only be created when that value is ≥ 1, we can also deduce that the risk range of any alarm will be from 1 to 10.

By using [Dsiem startup parameter](commands.md#dsiem-command-flags), you can then configure when will an alarm be labeled as Low, Medium or High risk based on that range. The default thresholds are:
- Low risk ⟶ risk value of 1 to 2
- Medium risk ⟶ risk value of 3 to 6
- High risk ⟶ risk value of 7 to 10 

### Example Directive to Alarm Processing

Let's say Dsiem is configured with the above `Ping Flood from SRC_IP` directive, and an `assets_*.json` that defines an asset value of 2 for `10.0.0.0/8` network.

When Dsiem receives the following series of events:
1. plugin_id: 1001, plugin_sid: 2100384, protocol: ICMP, src_ip: 10.0.0.1, dst_ip: 10.0.0.2
1. plugin_id: 1001, plugin_sid: 2100384, protocol: ICMP, src_ip: 10.0.0.1, dst_ip: 10.0.0.3
1. plugin_id: 1001, plugin_sid: 2100384, protocol: ICMP, src_ip: 10.0.0.2, dst_ip: 10.0.0.1
1. plugin_id: 1001, plugin_sid: 2100384, protocol: ICMP, src_ip: 10.0.0.1, dst_ip: 10.0.0.4
1. plugin_id: 1001, plugin_sid: 2100384, protocol: ICMP, src_ip: 10.0.0.1, dst_ip: 10.0.0.5 (repeated 10x)

The following processing will take place:

* On 1st event:
  * the event matches directive 1st rule condition because:
    * they have the same value for`plugin_id`, `plugin_sid`, and `protocol`.
    * the event `src_ip` (10.0.0.1) matches the directive `from` condition of `HOME_NET`. This is because `HOME_NET` refers to any addresses defined in `assets_*.json` file.
    * the event `dst_ip` (10.0.0.2) matches the directive `to` condition of `ANY`. This is because `ANY` will match all possible value.
  * backlog #1 is created to track this potential alarm. Its initial risk value will be `risk = (reliability * priority * asset value)/25 = (1 * 3 * 2)/25 = 0.24`, so no alarm will be created yet.
  * backlog #1 starts monitoring for incoming events that would trigger the 2nd stage rule condition.
* On 2nd event:  
  * the event matches backlog #1 2nd rule condition because:
    * the event `src_ip` matches the directive `from` value of `:1`. That `:1` denotes a reference to the `from` value matched in the first rule, which in this case is `10.0.0.1`.
    * backlog #1 enters 2nd stage with an initial risk value of `risk = (6 * 3 * 2)/25 = 1.44`. **This will generate a Low risk alarm**.
* On 3rd event:
  * the event doesn't match backlog #1 2nd rule condition because of the mismatch in `from` condition and the event's `src_ip`.
  * since this event does match the directive 1st rule, a new backlog #2 will be created to track it separately from backlog #1.
* On 4th event:
  * the event match backlog #1 2nd rule condition. This will raise its event count from 0 to 1 for the 2nd stage, and reset its internal timeout timer.
  * the event doesnt match backlog #2 2nd rule condition.
* On 5th to 15th event:
  * 5th to 7th events match backlog #1 2nd rule condition, and this causes the event count to reach the rule `occurrence` limit. Backlog #1 will then be moved to the 3rd stage with an initial risk value of `risk = (8 * 3 * 2)/25 = 1.92`. This is rounded up to 2, and results in an update of the associated alarm's risk label from Low to **Medium**.
  * the 8th to 15th events match backlog #1 3rd rule condition, and each will  increase the backlog's event count and reset its internal timeout timer.
  * None of these events match backlog #2 2nd rule condition.

Then if no more matching event is found, backlog #2 will expires and be removed after 10 minutes (600s), while backlog #1 will expires in 1 hour (3600s).