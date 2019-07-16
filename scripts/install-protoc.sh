#!/bin/bash
set -ex
pushd ~/
PROTOC_VER=3.8.0
PROTOC_ZIP=protoc-$PROTOC_VER-linux-x86_64.zip
curl -OL https://github.com/google/protobuf/releases/download/v$PROTOC_VER/$PROTOC_ZIP
sudo unzip -o $PROTOC_ZIP -d /usr/local bin/protoc
rm -f $PROTOC_ZIP
popd
