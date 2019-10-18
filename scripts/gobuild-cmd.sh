#!/bin/sh

command -v go >/dev/null || { echo 'cannot find go command in $PATH'; exit 1; }

cmd=${1}

# cgoflag should be 0 or 1
cgoflag=${2}
xtraflag=${3}
[ "$cgoflag" = "" ] && cgoflag=0

[ -z $cmd ] && cmd=$(find ./cmd/ -maxdepth 1 ! -path ./cmd/ -type d)

ver=$(git describe --tags)
now=$(date --utc --iso-8601=seconds)

for c in $cmd; do
  [ ! -d $c ] && echo $c directory doesnt exist, skipping. && continue
  echo building $c ver=${ver} buildtime=${now}
  GOFLAGS="-mod=vendor" CGO_ENABLED=${cgoflag} GOOS=linux GOARCH=amd64 go build ${xtraflag} -a -ldflags "-s -w -X main.version=${ver} -X main.buildTime=${now} -extldflags '-static'" $c
done

