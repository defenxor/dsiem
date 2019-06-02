This directory contains example ES index administration scripts to maintain 
siem_alarms when logstash is configured to write to a single index/alias, which
doesn't require the use of Elasticsearch filter plugin and the extra 
perm_index field. This scenario is not the default dsiem configuration, but may
still be useful in some cases due to the lower overhead during indexing time.

The scripts are:

- idx_use_alias.sh: to convert siem_alarms index to alias with the same name 
  that points to siem_alarms-current index.
- idx_merge_current.sh: to safely perform force_merge on siem_alarms-current 
  index.
- idx_create_daily.sh: to move old documents in siem_alarms-current to 
  siem_alarms-YYYY.MM.DD index.

