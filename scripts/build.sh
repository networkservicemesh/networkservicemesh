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

test -z $(go fmt ./...) || (echo "Run go fmt ./... and recommit your code";exit 1)
go get -u github.com/gogo/protobuf/protoc-gen-gogo
go generate ./...
go install ./...
go test $RACE_ENABLED ./...
