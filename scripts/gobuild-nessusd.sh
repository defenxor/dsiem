#!/bin/bash
ver=$(git describe --tags)
now=$(date --utc --iso-8601=seconds)
echo building ver=${ver} buildtime=${now}
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -ldflags "-X main.version=${ver} -X main.buildTime=${now} -extldflags '-static'" ./cmd/nessusd
