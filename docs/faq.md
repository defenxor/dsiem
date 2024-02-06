# Frequently Asked Questions

Here's the answers to the questions (technical or otherwise) that are frequently asked regarding this project.

## Why not just use OSSIM?

We did, and for many years. But as our use of Elastic stack grows, the necessity to parse and ingest logs into two different systems became more of an issue for the operation team.

Other than that, we also have concerns over OSSIM's future development as ~~Alienvault~~ AT&T Cybersecurity shifts their focus increasingly towards cloud-based offerings.

## But... why OSSIM?

Dsiem uses "OSSIM-style" event normalization scheme, correlation rules, and risk calculation method because that's what our analysts are most familiar with, and what we have developed most of our custom rules in.

This does mean that Dsiem carries over some of OSSIM design limitations, like the requirement to have an IPv4 address on normalized events, or limited number of custom data fields. For now at least, we accept those trade-offs for the easier transition and quicker adoption in our team.

## But it doesn't behave like OSSIM!

Well, that's why we don't call it "OSSIM-clone" 😉.

Two important differences mentioned were how Dsiem and OSSIM deal with vulnerability information, and how each structures its correlation rules. These are explained in more detail [here](./differences_from_ossim.md).

## Why there's no support for [insert feature here]?

Whatever additional functionalities you think should be in Dsiem, they're probably be best implemented outside of it. Dsiem is meant to replace just the OSSIM _correlation engine_ and not the entire OSSIM package which also includes a ticketing system, knowledge-base system, integrated security tools (Suricata, Ossec, OpenVAS, etc.), and PHP-based web interface.

To further illustrate this point, the only reason there's a web UI for Dsiem is because Kibana doesn't have a widget to do two things that we need: updating a document's field (alarm's `status` and `tags`), and presenting a many-many relationship (between `siem_alarms` and `siem_events` indices, through `siem_alarm_events`).

## Elastic already has its own SIEM, why should I use Dsiem?

Dsiem, OSSIM, and other similar systems that produce alarms based on predefined correlation rules have their own advantages in certain areas over the more flexible and ad hoc approach that systems like Elastic SIEM have.

For instance, the quantity of alarms produced by correlation rules will always be many orders of magnitude fewer than the number of events coming into the system. An alarm-based system allows security monitoring team to have a measurable baseline performance target like, "all alarms shall be investigated within 30 minutes timeframe". That kind of coverage commitment will be harder to formulate when the team have to deal directly with all the raw events coming into the system.

That said, there's no reason not to use both approaches if you have the resources for it. At Defenxor we use correlation rules to deal with known and repetitive threats, and the more flexible approach that Kibana and Elastic SIEM offer to look for patterns that could potentially be used as new rules.

## Why alarms have to flow from Dsiem → Filebeat → Logstash → Elasticsearch? Can't you send directly to ES?

Filebeat is used so Dsiem doesn't have to deal with network related errors or congestion in Logstash pipeline. Logstash is used so we can transform data as much as possible using its filter configuration instead of having to code them inside Dsiem. Elasticsearch API and client libraries changes frequently, so sending alarms directly to Elasticsearch may requires us to maintain a specific Dsiem version for each version of Elasticsearch.

This design also decouples Dsiem code from Elastic stack specific libraries, effectively allowing Dsiem to be used as an OSSIM-style correlation engine for non ELK stack if needed.

## So how to use Dsiem without Elastic stack?

First use something else other than Logstash for normalizing your logs in accordance to Dsiem [normalized event specification](https://github.com/defenxor/dsiem/blob/master/docs/dsiem_plugin.md#normalized-event). For instance, you can use [Fluentd](https://docs.fluentd.org/input) for this purpose.

Next, send those normalized events to Dsiem through HTTP. Again, should be possible with something like Fluentd [HTTP output](https://docs.fluentd.org/output/http).

Finally, substitute Filebeat with something else to read Dsiem output (`siem_alarms.json`), and send it to the final storage or notification destination. In Fluentd this may involve the `tail` input and `json` parser plugins sending results to one of Fluentd [data output plugin](https://www.fluentd.org/dataoutputs).

## Has this been used in production? Where is the stable version?

We use Dsiem in production to deliver our managed security monitoring service. Specifically, we use the latest [docker image](https://hub.docker.com/r/defenxor/dsiem) on all of our deployments inside a self-hosted Kubernetes cluster, and auto update them using [Keel](https://keel.sh/docs/) policy. This puts an extra pressure on our team to always make sure that the master branch is deployable and free of known error.

It's also worth noting the fact that Dsiem doesn't have to insert itself in the middle of the main ELK log ingestion pipeline. It only needs to tap into the pipeline (illustrated [here](https://github.com/defenxor/dsiem/blob/master/docs/event_processing.md#event-processing-flow)), so a proper Logstash configuration using [pipeline-to-pipeline](https://www.elastic.co/guide/en/logstash/current/pipeline-to-pipeline.html) or multiple Logstash instances will be able to shield the main log ingestion flows in case anything breaks in Dsiem.

## Is there a TL;DR version to evaluate?

We have a full working demo for [docker compose](../demo) or [virtual machine](../demo/vagrant) environment. You will still have to install a couple of requirements like Docker and Docker Compose, or VirtualBox and Vagrant -- but beyond that everything should be pretty much automated.

## The web UI lacks features and aesthetic!

You _really_ should be monitoring Dsiem alarms and events mainly from Kibana. You then open Dsiem web UI only when you want to close an alarm, change its tag, or to see the parent-child relationship between alarm and the corresponding events. To easily adopt this workflow, Dsiem default Kibana dashboard has a [scripted field](https://www.elastic.co/guide/en/kibana/7.3/scripted-fields.html) that links each alarm to their respective web UI's page.

If that's not good enough, you can always just use `curl` command against `siem_alarms` index to update the alarm's `tag` and `status` directly 😎. Example on how to change alarm ID q59Azehjpp status to Closed:

```shell
$ curl -X POST "localhost:9200/siem_alarms/_update_by_query?pretty" -H 'Content-Type: application/json' -d'
{
  "script": {
    "source": "ctx._source.status = \"Closed\"",
    "lang": "painless"
  },
  "query": {
    "term": {
      "_id": "q59Azehjpp"
    }
  }
}
'
```

## What about the server's performance?

If Dsiem is unable to keep up with your log ingestion rate, i.e. you're experiencing processing delays, then likely you need to allocate more resources to it. Try adding more backend nodes and distribute your directives among them.

If instead you think Dsiem is already using too much resource, understand that we put processing throughput high in our priority and that may well end up consuming a lot of resources (particularly CPU) when you have many directives defined.

Read [here](./managing_performance.md) for more suggestions on how to optimise Dsiem performance.

## How to access the Docker Compose example from remote?

If you followed the docker compose installation guide, then by default Dsiem web UI will only work from `localhost`. Extra steps are needed to be able to access it remotely.

Alternative #1: Use SSH tunnel to access those ports

- Setup appropriate SSH access from your local machine to the remote server where the Docker daemon is running
- Do this from your local machine:
  ```shell
  $ ssh $remote_server -L 8080:localhost:8080 -L 9200:localhost:9200 -L 5601:localhost:5601
  ```
- Open a browser in your local machine, then access Dsiem UI from http://localhost:8080/ui/, Elasticsearch from http://localhost:9200/, and Kibana from http://localhost:5601/.

Alternative #2: Adjust the `docker-compose.yml` configuration

- The first step is to make sure you can indeed access the docker-hosted Elasticsearch and Kibana remotely. From your local machine, try opening http://your-server-ip:9200/ and http://your-server-ip:5601/ using a browser. Investigate your network setup and docker daemon config if those URLs don't work.

- Open `docker-compose.yml` in an editor, and search for these lines.

  ```yaml
  - DSIEM_WEB_ESURL=http://localhost:9200
  - DSIEM_WEB_KBNURL=http://localhost:5601
  ```

  Replace `localhost` above with the actual server IP address or hostname that is accessible from the browser.

- Then search for this line in the same file above:

  ```yaml
  - http.cors.allow-origin=/https?:\/\/localhost(:[0-9]+)?/
  ```

  and replace it with:

  ```yaml
  - http.cors.allow-origin=*
  ```

- Refresh Dsiem container:
  ```shell
  $ docker-compose up -d
  ```
- Access Dsiem web UI from http://your-server-ip:8080/ui/

## How to restrict access to and/or lockdown the web UI?

The web UI is just an Angular app that runs completely on the browser and access alarms data stored in Elasticsearch. It has no dependency to the `dsiem` binary, nor does it access any API or data from it.

The only "integrations" that we have for the web UI are:

- By-default, `dsiem` will serve the web UI files from its `/ui/` HTTP endpoint for easier deployment. This of course, can just as easily be done using something like Apache or Nginx.
- The web UI reads ES and Kibana endpoint locations from `/ui/assets/config/esconfig.json`. Again, for easier deployment, in the docker image we have a [start-up script](https://github.com/defenxor/dsiem/blob/master/deployments/docker/build/s6files/cont-init.d/01_set_web_esaddr) that automatically write the content of that file based on `DSIEM_WEB_ESURL` and `DSIEM_WEB_KBNURL` environment variables. For non-docker environment, you can just change the file content manually by editing `${DSIEM_DIR}/web/dist/assets/config/esconfig.json`.

So to answer the original question: No, you don't need to restrict access to the web UI itself. For access control purposes, the web UI has the same access level as a `curl` command running on the same machine.

You _do_ however, have to restrict access to Elasticsearch and Kibana. You'll also have to restrict access to the `esconfig.json` file above if you put something like `http://user:password@elasticsearch-server:9200` in it either manually or through `DSIEM_WEB_ESURL` environment variable for authentication purpose (see next question for more).

## How to use the web UI to access an X-Pack Security-enabled Elastic cluster?

The web UI needs access to Elasticsearch endpoint to read alarms data, and (optionally) to Kibana for pivoting to it from the web UI's detail alarm view.

For Elasticsearch, first make sure you create a dedicated user account for Dsiem web UI, and only authorize it to access `siem_alarms`, `siem_events`, and `siem_alarm_events` indices.

After that, you will need to enter that credential into `${DSIEM_DIR}/web/dist/assets/config/esconfig.json` file (notice the `${UID}` and `${PASSWD}` placeholders):

```json
{
  "elasticsearch": "http(s)://${UID}:${PASSWD}@address-reachable-from-the-browser:9200/",
  "kibana": "http(s)://address-reachable-from-the-browser:5601/"
}
```

Elasticsearch CORS configuration should also be adjusted as follows:

```
    http.cors.enabled=true
    http.cors.allow-credentials=true
    http.cors.allow-origin=*
    http.cors.allow-headers=Authorization,X-Requested-With,Content-Type,Content-Length
```

> [!TIP]
> Example of how to use Dsiem with authentication-enabled Elastic stack is given in [this docker compose file](https://github.com/defenxor/dsiem/blob/master/deployments/docker/docker-compose-basic-auth.yml). It's using the default `elastic` account everywhere (including in the Logstash config files), but it should be easy to use multiple dedicated accounts instead.

_Don't forget to also restrict access to `esconfig.json` from the web with something like Nginx/Apache or similar frontend that does authentication and authorization._

Kibana on the other hand uses form-based authentication, so the above method will not work. The workaround for now is to first login into it manually from a separate browser tab, before attempting to click a link to it from the web UI.

## How to trigger alarm only during a specific time window?

The most efficient way to achieve this is by only sending normalized events in question to Dsiem during that time frame. So outside of that range, Elasticsearch will still receive the events normally in `siem_events` index but Dsiem will not.

We provide a Logstash ruby filter script to achieve that purpose [here](https://github.com/defenxor/dsiem/blob/master/deployments/docker/conf/logstash/scripts/allow_within_timerange.rb). For instance, to send normalized events with `plugin_id` 1001 and `plugin_sid` 12345 to Dsiem _only_ between 23.00 - 2.00 (UTC), you can put the following `filter` configuration in Logstash:

```
filter {
  if [@metadata][siem_data_type] == "normalizedEvent" {
    if [plugin_id] == 1001 and [plugin_sid] == 12345 {
      ruby {
        path => "/replace/with/full/path/to/allow_within_timerange.rb"
        script_params => {
          "timestamp_field" => "timestamp"
          "time_from" => "23:00"
          "time_to" => "02:00"
        }
      }
    }
  }
}
```

The above should be placed right before Logstash `output` to Dsiem. In the example Logstash config file that location would be [here](https://github.com/defenxor/dsiem/blob/2b8c7dbfd18f852092ddd7a007a25ce0c02ba903/deployments/docker/conf/logstash/conf.d/80_siem.conf#L13).
