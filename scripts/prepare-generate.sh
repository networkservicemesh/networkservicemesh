#!/usr/bin/env sh

go install k8s.io/code-generator/cmd/deepcopy-gen
go install github.com/golang/protobuf/protoc-gen-go
go get golang.org/x/tools/cmd/stringer

