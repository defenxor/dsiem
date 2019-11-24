#!/bin/bash

scriptdir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
cd $scriptdir

git checkout ../dpluger/70_dsiem-plugin_suricata.conf
rm -rf ../dpluger/directives_dsiem.json ../dpluger/*tsv ../docker/conf/dsiem/configs/directives_dsiem.json
