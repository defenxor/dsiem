#!/bin/bash

# cd web/ui
cd web/ui
ng build --prod --build-optimizer --base-href /ui/
rm -rf ../dist
cp -r ./dist ../
