#!/bin/bash
#
# This script builds the networkservicemesh
#

set -e

[ -d vendor/github.com/ligato/networkservicemesh/ ] && (echo "Run: rm -rf vendor/github.com/ligato/networkservicemesh;dep ensure";exit 1)
test -z $(go fmt ./...) || (echo "Run go fmt ./... and recommit your code";exit 1)
go get -u github.com/golang/protobuf/protoc-gen-go
go generate ./...
go install ./...
go test ./...
