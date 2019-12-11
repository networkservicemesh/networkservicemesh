#!/bin/bash

pushd "$(dirname "${BASH_SOURCE[0]}")" || exit 1
AWS_REGION=us-east-2 go run ./... DeleteAll "$1" "$2"
popd || exit 0