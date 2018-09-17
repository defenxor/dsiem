TSV parser for Go
====

[![Build Status](https://travis-ci.org/dogenzaka/tsv.svg?branch=master)](https://travis-ci.org/dogenzaka/tsv)
[![Coverage Status](https://coveralls.io/repos/dogenzaka/tsv/badge.svg)](https://coveralls.io/r/dogenzaka/tsv)
[![License](http://img.shields.io/badge/license-MIT-red.svg?style=flat)](https://github.com/dogenzaka/rotator/blob/master/LICENSE)

tsv is tab-separated values parser for GO. It will parse lines and insert data into any type of struct. tsv supports both simple structs and structs with tagging.

```
go get github.com/dogenzaka/tsv
```

Quickstart
--

tsv inserts data into struct by fields order.

```go

import (
    "fmt"
    "os"
    "testing"
    )

type TestRow struct {
  Name   string // 0
  Age    int    // 1
  Gender string // 2
  Active bool   // 3
}

func main() {

  file, _ := os.Open("example.tsv")
  defer file.Close()

  data := TestRow{}
  parser, _ := NewParser(file, &data)

  for {
    eof, err := parser.Next()
    if eof {
      return
    }
    if err != nil {
      panic(err)
    }
    fmt.Println(data)
  }

}

```

You can define tags to struct fields to map values.

```go
type TestRow struct {
  Name   string `tsv:"name"`
  Age    int    `tsv:"age"`
  Gender string `tsv:"gender"`
  Active bool   `tsv:"bool"`
}
```

Supported field types
--

Currently this library supports limited fields

- int
- string
- bool

