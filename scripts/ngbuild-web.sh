#!/bin/sh

command -v npm >/dev/null || { echo 'cannot find npm command in $PATH'; exit 1; }

dir="ui"
[ "$1" = "internal" ] && dir="${dir}-internal"
cd web/$dir
npm install
npm run ng -- build --prod --build-optimizer --base-href /ui/
rm -rf ../dist
cp -r ./dist ../
