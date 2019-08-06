#!/bin/bash
echo "WARNING: Use this script only if you know what you're doing!!!"
echo "You likely want to use scripts/prepare-generate.sh instead."
echo "- protoc version <= 3.9.0 is missing UnimplementedServer* method generation."

set -ex
pushd ~/
PROTOC_VER=3.8.0
PROTOC_ZIP=protoc-$PROTOC_VER-linux-x86_64.zip
curl -OL https://github.com/google/protobuf/releases/download/v$PROTOC_VER/$PROTOC_ZIP
sudo unzip -o $PROTOC_ZIP -d /usr/local bin/protoc
rm -f $PROTOC_ZIP
popd
