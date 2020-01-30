#!/usr/bin/env sh

tmp="$(mktemp -d)"
cd "${tmp}" || exit 1
go get k8s.io/code-generator/cmd/deepcopy-gen@v0.17.2
go get github.com/golang/protobuf/protoc-gen-go@v1.3.3
go get golang.org/x/tools/cmd/stringer@v0.0.0-20200130002326-2f3ba24bd6e7
rm -rf "${tmp}"