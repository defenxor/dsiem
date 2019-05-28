#!/bin/bash

go=$(which go)
npm=$(which npm)

if [ ! -z $go ]; then
  go get -v github.com/mmta/addlicense && \
  addlicense -c "PT Defender Nusa Semesta and contributors" -l gpl -p Dsiem ./internal ./pkg ./cmd 
  echo done processing go files.
else
  echo go command not available, skipping go code.
fi

if [ ! -z $npm ]; then
 year=$(date +"%Y")
 sed -i "s/20[0-9][0-9]/$year/" web/ui/LICENSE.txt 
 cd web/ui && \
 npm install >/dev/null 2>/dev/null && \
 npm run licchkadd 
else
  echo npm command not available, skipping js code.
fi
