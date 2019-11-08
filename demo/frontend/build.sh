#!/bin/bash

rm -rf web/node_modules web/dist
docker build -t defenxor/dsiem-demo-frontend .
