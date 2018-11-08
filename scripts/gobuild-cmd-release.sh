#!/bin/bash

ver=${1}
cmd=${2}
goos=${3}

[ -z $cmd ] && cmd=$(find ./cmd/ -maxdepth 1 ! -path ./cmd/ -type d 2>/dev/null)
[ -z $goos ] && goos="linux windows darwin"
[ -z $ver ] && ver=$(git describe --tags 2>/dev/null)

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
    GOFLAGS="-mod=vendor" CGO_ENABLED=0 GOOS=$os GOARCH=amd64 go build -a -ldflags "-s -w -X main.version=${ver} -X main.buildTime=${now} -extldflags '-static'" -o $bdir/$n $c
  done
  mkdir -p $bdir/web/dist && cp -r ./web/dist/* $bdir/web/dist/
  cp -r ./configs ./LICENSE ./README.md $bdir/
  cd $bdir 
  zip -9 -r $rdir/dsiem-server_$os_amd64.zip dsiem configs web LICENSE README.md
  tools=$(ls | grep -v dsiem | grep -v configs | grep -v web)
  zip -9 $rdir/dsiem-tools_$os_amd64.zip $tools
  cd $curdir
  rm -rf $bdir
done
cd $rdir
for f in $(ls $rdir | grep -v sha256); do
  sha256sum $f > $f.sha256
done
