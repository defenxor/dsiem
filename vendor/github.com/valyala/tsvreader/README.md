# tsvreader - fast reader for tab-separated data

## Features

* Optimized for speed. May read more than 20M rows per second on a single
  CPU core.
* Compatible with `TSV` (aka `TabSeparated`) format used in [ClickHouse](https://github.com/yandex/ClickHouse) responses.
  See [chclient](https://github.com/valyala/chclient) - clickhouse client built on top of `tsvreader`.
* May read rows with variable number of columns using [Reader.HasCols](https://godoc.org/github.com/valyala/tsvreader#Reader.HasCols).
  This functionality allows reading [WITH TOTALS](http://clickhouse.readthedocs.io/en/latest/reference_en.html#WITH+TOTALS+modifier)
  row from `ClickHouse` responses and [BlockTabSeparated](http://clickhouse.readthedocs.io/en/latest/reference_en.html#BlockTabSeparated)
  responses.

## Documentation

See [these docs](https://godoc.org/github.com/valyala/tsvreader).
