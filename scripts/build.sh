#!/bin/bash
#
# This script builds the networkservicemesh
#

export RACE_ENABLED="-race"
while test $# -gt 0; do
	case "$1" in
		--race-test-disabled)
			export RACE_ENABLED=""
			;;
		*)
			break
			;;
	esac
	shift
done

[ -d vendor/ligato/networkservicemesh/ ] && (echo "Run: rm -rf vendor/ligato/networkservicemesh;dep ensure";exit 1)
test -z $(go fmt ./...) || (echo "Run go fmt ./... and recommit your code";exit 1)
go get -u github.com/golang/protobuf/protoc-gen-go
go generate ./...
go install ./...
go test $RACE_ENABLED ./...
