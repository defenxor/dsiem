# Dpluger tool demo

This page illustrates how to use `dpluger` to generate Dsiem directives using a `dpluger_config.json` configured for suricata index. The same workflow can be used for other indices too.

## Steps

First generate attack traffic to populate suricata index. Do
something like this from a remote machine:

```shell
$ sqlmap --batch -u http://[targetip]:8081/cgi-bin/
```

Generate `suricata_plugin-sids.tsv` using the following command (will also generate
Logstash config for suricata, but that's already done beforehand in the demo env.):

```shell
$ dpluger run
```

Create suricata directive file based on that:

```shell
$ dpluger directive -f suricata_plugin-sids.tsv
```

Activate those directives in dsiem:

```shell
$ cp directives_dsiem.json ../docker/conf/dsiem/configs/directives_dsiem.json
$ docker restart dsiem
```

Now trigger the same attack as before (e.g. sqlmap, etc.), and notice how
Dsiem web UI and Kibana dashboard will now display new alarms based on those
directives.

The commands above starting from `dpluger run` can be repeated as needed to add
more directives as more types of events occur in the source (suricata) index.
