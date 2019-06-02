# Dsiem Elasticsearch Indices

Dsiem uses 3 types of elasticsearch index:

- `siem_alarms` : to store [alarms](./directive_and_alarm.md) documents.
- `siem_events` : to store [normalized events](./dsiem_plugin.md#normalized-event) documents.
- `siem_alarm_events` : to express many-to-many relationship between the document IDs in `siem_alarms` and `siem_events`.

All indices are created by Logstash using a one-per-day index pattern (i.e. `siem_events-YYYY.MM.DD`), with important implementation note describe below.

## Implementation Note

Documents in `siem_events` and `siem_alarm_events` are meant to be written only once and never have to be updated. This makes them an ideal target for one-per-day index pattern.

On the other hand, documents in `siem_alarms` are typically updated multiple times. Users may update the alarm status and/or conclusion, and Dsiem event correlation process may increase the alarm risk as more incoming events match the rules condition. A straight forward one-per-day indexing pattern in this case will results in a single alarm ID being stored in multiple indices. This is true for any alarm created prior to an index rollover time and kept being updated after it.

To overcome that, the Logstash configuration for `siem_alarms` uses Elasticsearch filter plugin to find the index that contain the same document ID with the one being indexed. If there is match Logstash will update that index instead of today's index, so the duplicate IDs issue can be avoided. As far as we know, this method requires the least manual administration effort compared to other alternatives, but does cost an extra query to Elasticsearch and an extra field on the `siem_alarms` document for storing the index location (unfortunately Elasticsearch filter plugin can only search fields within the `_source` field, so we can't use the built-in `_index` field here).
