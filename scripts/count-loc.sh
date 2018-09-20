#!/bin/bash
wc -l $(find ./ -name "vendor" -prune -o -name "*.go" -print)
