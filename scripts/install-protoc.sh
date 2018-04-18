#!/bin/bash
set -ex
pushd ~/
wget https://github.com/google/protobuf/releases/download/v3.5.1/protoc-3.5.1-linux-x86_64.zip
unzip protoc-3.5.1-linux-x86_64.zip

echo "GOPATH=${GOPATH}"
echo "GOBIN=${GOBIN}"
ls ${GOPATH}
ls ${GOBIN}
echo "Checking ${GOPATH}/bin"
ls ${GOPATH}/bin
popd
