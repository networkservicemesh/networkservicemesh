#!/bin/bash


pushd "$(dirname "${BASH_SOURCE[0]}")"/../../test/cloudtest/pkg/providers/packet/packet_cleanup || exit 1
go run ./... -t "$PACKET_AUTH_TOKEN" -p "$PACKET_PROJECT_ID" -k y -c y
popd || exit 0