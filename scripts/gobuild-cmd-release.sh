#!/bin/bash

cmd=${1}
goos=${2}

[ -z $cmd ] && cmd=$(find ./cmd/ -maxdepth 1 ! -path ./cmd/ -type d 2>/dev/null)
[ -z $goos ] && goos="linux windows darwin"

ver=$(git describe --tags 2>/dev/null)
now=$(date --utc --iso-8601=seconds)
[ -z $ver ] && ver="untagged"

echo "target OS: $goos;" commands to build: $cmd
curdir=$(pwd)
rdir=$curdir/temp/release/$ver
mkdir -p $rdir
for os in $goos; do
  echo "** building for $os **"
  bdir=./temp/build/$os
  rm -rf $bdir && mkdir -p $bdir
  for c in $cmd; do
    [ ! -d $c ] && echo $c directory doesnt exist, skipping. && continue
    n=$(basename $c)
    [ "$os" == "windows" ] && n="${n}.exe"
    echo building $c ver=${ver} buildtime=${now} for $os ..
    CGO_ENABLED=0 GOOS=$os GOARCH=amd64 go build -a -ldflags "-s -w -X main.version=${ver} -X main.buildTime=${now} -extldflags '-static'" -o $bdir/$n $c
  done
  mkdir -p $bdir/web/dist && cp -r ./web/dist/* $bdir/web/dist/
  cp -r ./configs $bdir/
  cd $bdir 
  if [ "$os" == "linux" ]; then
    zip -9 -r $rdir/dsiem-server-$os-amd64.zip dsiem configs web
  fi
  tools=$(ls | grep -v dsiem | grep -v configs | grep -v web)
  zip -9 $rdir/dsiem-tools-$os-amd64.zip $tools
  cd $curdir
  rm -rf $bdir
done

