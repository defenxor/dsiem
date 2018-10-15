#!/bin/bash

dir="./deployments/docker"
files="dsiem configs web/dist"

[ ! -e $dir ] && echo must be executed from app root directory. && exit 1
curdir=$(pwd)

./scripts/gobuild-cmd.sh ./cmd/dsiem

cleanexit() {
  cd $curdir
  echo cleaning up and exiting.
  for f in $files; do
    rm -rf $dir/$(basename $f)
  done 
  exit $1
}

for f in $files; do 
  cp -r ./$f $dir/ || (echo cannot copy $f to $dir && cleanexit 1)
done 
cd $dir
ver=$(git describe --tags)
docker build -f Dockerfile -t dsiem:$ver -t dsiem:latest .
cleanexit $?

