#!/bin/bash
if [ "$1" == "all" ]; then
go test `go list ./... | grep -v temp`
fi 
if [ "$1" == "update" ]; then
  go test -cover ./internal/dsiem/pkg/event -coverprofile=./test/profile.out -update
else
  go test -cover ./internal/dsiem/pkg/event -coverprofile=./test/profile.out
fi

if [ "$1" == "html" ]; then go tool cover -html=./test/profile.out; fi

