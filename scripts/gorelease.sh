#!/bin/bash

export GITHUB_REPO="dsiem"
export GITHUB_USER="defenxor"

[ "$1" == "" ] && echo need semver version as 1st argument, and optional pre-release flag as 2nd argument. Example $0 v0.1.0 pre && exit 1
ver="$1"
[ "$2" == "pre" ] && pre="-p"

read -p "This will create a git tag and release for $1. Are you sure? " -n 1 -r
echo 
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    [[ "$0" = "$BASH_SOURCE" ]] && exit 1 || return 1 
fi

git tag -a $ver

./scripts/gobuild-cmd-release.sh || (echo failed to build release files && exit 1)

github-release release -t $ver $pre || (echo failed to create github release && exit 1)

for f in $(ls ./temp/release/$ver); do
  github-release upload -t $ver -f ./temp/release/$ver/$f -n $f
done

