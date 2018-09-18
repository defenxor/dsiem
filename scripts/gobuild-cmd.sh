#!/bin/bash

cmd=${1}

[ -z $cmd ] && cmd=$(find ./cmd/ -maxdepth 1 ! -path ./cmd/ -type d)

ver=$(git describe --tags)
now=$(date --utc --iso-8601=seconds)

for c in $cmd; do
  [ ! -d $c ] && echo $c directory doesnt exist, skipping. && continue
  echo building $c ver=${ver} buildtime=${now}
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -ldflags "-s -w -X main.version=${ver} -X main.buildTime=${now} -extldflags '-static'" $c
done

