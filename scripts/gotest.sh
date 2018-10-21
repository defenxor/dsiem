#!/bin/bash
if [ "$1" == "" ]; then
go test -cover $(go list ./... | grep -v temp)
exit 
fi 
if [ "$1" == "update" ]; then
  go test -cover ./internal/dsiem/pkg/event -coverprofile=./test/profile.out -update
fi

if [ "$1" == "html" ]; then go tool cover -html=./test/profile.out; fi

