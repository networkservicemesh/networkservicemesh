#!/bin/bash


pushd "$(dirname "${BASH_SOURCE[0]}")"/../../test/cloudtest/pkg/providers/packet/packet_cleanup || exit 1
go run ./... -k y -c y
popd || exit 0