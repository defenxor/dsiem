# Managing Performance

Dsiem is designed to scale horizontally, which means we can add more nodes and distribute correlation rule directives among them to cope with increasing demands.

In practice however, hardware and network resources are often limited, and we will have to work with what's available at hand.

This page gives several suggestions on how to deploy Dsiem with performance consideration in mind. In addition, tips on how to detect performance issue are also given at [the end of this page](#Detecting-performance-issue).

## Selectively ingest logs from Logstash

It makes no sense to send a log to Dsiem if you don't have correlation rules for it.

You can avoid sending unnecessary logs by reconfiguring the Logstash filter that generates the normalized events. Similar effect can also be achieved by supplying an appropriate filter in `dplugin_config.json` before running `dpluger run` command to create the Logstash plugin.

## Distribute directives to multiple nodes (and hardware whenever possible)

The typical Logstash ingestion pipeline(s) will always process events faster than Dsiem that has to correlate those events against X number of directives. The more directives you have on a single node, the more pronounced this effect will be. 

For instance, given the following backend nodes:
- backend-A, 1000 directives defined, 10 active backlogs in memory 
- backend-B, 100 directives defined, 1000 active backlogs in memory

Backend-A will have a harder time keeping up with the rate of incoming events compared to backend-B, even though most of its 1000 directives never actually match any of those events (hence its low number of active backlogs).

The above is true because in order to process things concurrently as much as possible, Dsiem copies each incoming event to an array of backlog managers, each of whom responsible for a single directive defined in the configuration files. So if you have 2000 directives defined, at runtime you will have 2000 backlog managers all waiting for the next event to process.

Those backlog managers will then have to compete for the system's limited CPU cores when processing incoming events. In a system handling 2000 events/sec, individual backlog managers will have to process *each* event in less than 500μs to avoid introducing delays. Having fewer directives reduces competition for CPU time, and therefore allows each directive to complete its processing within the time duration limit.

## Prioritise directives and allocate resources accordingly

Directives are not created equal. Directives that detect more severe consequences should be given higher priority and should receive more allocation compared to other directives.

You can treat directives differently by using a separate set of nodes (both frontend and backend) for each class of directives, and defining a separate overload coping strategy for each of them.

Dsiem offers two such strategies to select from:

1. Use a fixed length queue and discard new events when that queue is full

   The advantages of this strategy are:
   - Events that *do* get processed will have a recent timestamp.
   - Backend nodes will have a relatively constant and predictable resource usage.
   - NATS, Logstash, and frontend nodes do not have to adapt to backend nodes condition.

   The obvious (and rather severe) disadvantage of this is Dsiem will skip processing events from time to time.

   >**Note**: Use this strategy by setting `maxQueue` to a number higher than 0, and `maxDelay` to 0. The fixed queue length then will be set to `maxQueue`, and `maxDelay` = 0 will prevent frontend from throttling incoming events.

1. Use an unbounded queue and auto-adjust frontend ingestion rate (events/sec) to apply back-pressure to Logstash

   In this case whenever Dsiem backend nodes detect an event that has a timestamp older than the configured threshold, they will instruct frontends to reduce the rate of incoming events from Logstash. Frontends will gradually increase its ingestion rate again once the backends no longer report overload condition.

   Advantage of this strategy is that eventually all events will be processed in order.

   The disadvantages are:
    - There could be processing delays from time to time.
    - The processing delays may never go away if the log sources never reduce their output rate.
    - Sustained reduction of delivery rate from Logstash to frontends will cause Logstash to overflow its queue capacity, and depending on how it's configured, Logstash may end up stop receiving incoming events from its input. Using Logstash persistent queue backed by a large amount of storage space will not help either — in fact that may only worsen the processing delay issue.
    <p></p>

    >**Note**: Use this strategy by setting `maxQueue` to 0, and `maxDelay` to a number higher than 0. The queue length will then be unbounded, and `maxDelay` (seconds) will be used by backend to detect processing delay and report this condition to frontend, which will then apply back-pressure to Logstash.

   >_Processing delay_ occurs when the duration between the time that _an event was received by frontend_ to the time when _that event is processed by a directive_, is greater than `maxDelay`.

Now, for instance suppose that in a limited resource environment, you have 100 critical directives and 1000 lower priority directives both evaluating the same sources of logs. You want the critical directives to be applied to all events at all times, and to have a maximum processing delays of 5 minutes. In exchange for that, you're willing to let the lower priority directives occasionally skip events, as long as the alarms that they do manage to produce are based on recent enough events, which will make them at least relevant and still actionable.

Given that scenario, you can use the following strategy to make the best out of the situation:

- Use unbounded queue and Dsiem EPS rate auto-adjustment (with threshold set to 5 minutes delay) on a set of nodes that host the 100 critical directives. Make sure that the nodes have enough hardware resources allocated to cope with normal ingestion rate, so that the auto-adjustment will only be triggered sparingly during a temporary spike and will not last for long. 
At Logstash end, use persistent queue on the pipeline with enough capacity to prevent it from blocking its input during reduced output rate to Dsiem frontend. This last bit isn't necessary if the input is Filebeat, or similar producer that doesn't discard events when they can't send to Logstash.

- Use fixed length queue on a set of nodes that host the lower priority directives. They can run on the hardware that aren't being used by the nodes hosting critical directives.

## Shield the main Logstash ingestion pipeline

For production use, it's important to make sure that any Dsiem performance issues or downtime will not affect the main Logstash ingestion pipeline to Elasticsearch.

This can be implemented using Logstash [pipeline-to-pipeline](https://www.elastic.co/guide/en/logstash/current/pipeline-to-pipeline.html) feature or by running a cascade of Logstash instances configured in a certain way. To assist in this, `dpluger run` has a `--usePipeline` flag that will create plugins in a format that is more suitable for multiple pipeline configuration.

## Detecting performance issue

Dsiem regularly prints out information that can be used to detect performance-related issues.

The following shows an example of a node with a fixed length queue having problem keeping up with the inbound ingestion rate:

```shell
$ docker logs -f dsiem | jq ".msg" --unbuffered | grep -E '(queue|Watchdog|Single)'
"Backend queue length: 49231 dequeue duration: 2.145103ms timed-out directives: 0(0/0/0) max-proc.time/directive: 900µs"
"Single event processing took 2.145103ms, may not be able to sustain the target 1000 events/sec (1ms/event)"
"Backend queue discarded: 1308 events. Reason: FixedFIFO queue is at full capacity"
"Watchdog tick ended, # of backlogs: 425 directives (in-use/total): 30/1283"
"Watchdog tick ended, # of backlogs: 430 directives (in-use/total): 30/1283"
"Watchdog tick ended, # of backlogs: 419 directives (in-use/total): 30/1283"
"Backend queue length: 49973 dequeue duration: 330.378µs timed-out directives: 1(0/1/0) max-proc.time/directive: 900µs"
"Backend queue discarded: 2207 events. Reason: FixedFIFO queue is at full capacity"
"Watchdog tick ended, # of backlogs: 433 directives (in-use/total): 30/1283"
```
Those log lines show the following:
- There are around 49k events constantly in queue, and 1308 followed by 2207 events discarded because the queue is full.
- A single event processing time took around 300µs to 2ms, and that upper range is too long for the configured `maxEPS` parameter of 1000 events/sec (or 1ms max. processing time per event). This long processing time is what causing the queue to fill up and never had a chance to drain.
- The system has > 400 active backlogs, all created from just 30 of the 1283 directives defined.

Based on the above we can try to relieve the performance bottleneck by moving the rarely used directives to other nodes running on a different hardware. This change will reduce single event processing time and thereby preventing the queue from constantly being filled to its maximum capacity.

As a comparison, here's an example log output from a node that isn't experiencing performance problem:

```shell
$ docker logs -f dsiem | jq ".msg" --unbuffered | grep -E '(queue|Watchdog|Single)'
"Watchdog tick ended, # of backlogs: 1495 directives (in-use/total): 9/77"
"Backend queue length: 0 dequeue duration: 21.849µs timed-out directives: 0(0/0/0) max-proc.time/directive: 900µs"
"Watchdog tick ended, # of backlogs: 1491 directives (in-use/total): 9/77"
"Watchdog tick ended, # of backlogs: 1489 directives (in-use/total): 9/77"
"Watchdog tick ended, # of backlogs: 1489 directives (in-use/total): 9/77"
"Backend queue length: 0 dequeue duration: 143.692µs timed-out directives: 0(0/0/0) max-proc.time/directive: 900µs"
"Watchdog tick ended, # of backlogs: 1483 directives (in-use/total): 9/77"
"Watchdog tick ended, # of backlogs: 1479 directives (in-use/total): 9/77"
"Watchdog tick ended, # of backlogs: 1479 directives (in-use/total): 9/77"
```
Those log lines show that:
- The queue is never used at all. Single event processing time is around 21-143µs, still way faster than the configured limits of 900µs (or 90% of the `maxEPS` parameter of 1k/sec).
- The node is tracking almost 1500 active backlogs created from 9 directives (out of the total 77 directives defined), and that doesn't negatively affect its performance.

So for this particular node, we can try to increase its utilisation by moving more directives to it, or by increasing its incoming event ingestion rate.
