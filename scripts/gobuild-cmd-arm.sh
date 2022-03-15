#!/bin/sh

command -v go >/dev/null || { echo 'cannot find go command in $PATH'; exit 1; }

cmd=${1}

goos=${2}
[ -z $goos ] && goos=darwin

goarch=${3}
[ -z $goarch ] && goarch=arm64


[ -z $cmd ] && cmd=$(find ./cmd/ -maxdepth 1 ! -path ./cmd/ -type d)

ver=$(git describe --tags)
now=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

for c in $cmd; do
  [ ! -d $c ] && echo $c directory doesnt exist, skipping. && continue
  echo building $c ver=${ver} buildtime=${now} for $goos/$goarch
  GOFLAGS="-mod=vendor" CGO_ENABLED=${cgoflag} GOOS=$goos GOARCH=$goarch go build ${xtraflag} -a -ldflags "-s -w -X main.version=${ver} -X main.buildTime=${now} -extldflags '-static'" $c
done
