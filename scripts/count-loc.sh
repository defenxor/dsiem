#!/bin/bash
wc -l $(find ./ \( -name vendor -o -name test \) -prune -o -name "*.go" -print)
