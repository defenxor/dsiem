#!/bin/bash
wc -l $(find ./ \( -name vendor -o -name temp \) -prune -o -name "*.go" -print)
