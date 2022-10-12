#!/bin/bash

dir="./deployments/docker/build"
[ ! -e $dir ] && echo must be executed from app root directory. && exit 1
curdir=$(pwd)

cd $dir
ver=$(git describe --tags)
docker build -f Dockerfile -t defenxor/dsiem:$ver -t defenxor/dsiem:latest . --build-arg ver=$ver
