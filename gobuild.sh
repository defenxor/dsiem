#!/bin/bash

CGO_ENABLED=0 GOOS=linux GOARCH=386 go build -a -ldflags '-extldflags "-static"' -o siem
