#!/bin/sh
set -ex
wget https://github.com/google/protobuf/releases/download/v2.6.0/protobuf-2.6.0.tar.bz2
tar -xvjf protobuf-2.6.0.tar.bz2
cd protobuf-2.6.0 && ./configure --prefix=/usr && make && sudo make install
