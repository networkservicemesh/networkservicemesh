#!/bin/bash
#
# This script builds the networkservicemesh
#

go get -u github.com/golang/protobuf/protoc-gen-go
go generate ./...
go install ./...
go test ./...

