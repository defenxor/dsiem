#!/bin/bash
if [ "$1" == "" ]; then
go test -race -cover $(go list ./... | grep -v temp)
exit $?
fi 
if [ "$1" == "all" ]; then
go test -race -cover -count=1 $(go list ./... | grep -v temp)
exit $?
fi

if [ "$1" == "update" ]; then
  go test -cover ./internal/dsiem/pkg/event -coverprofile=./test/profile.out -update
  exit $?
fi

if [ "$1" == "html" ]; then go tool cover -html=./test/profile.out; fi

