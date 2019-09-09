#!/usr/bin/env sh

GO111MODULE=off go get -u k8s.io/code-generator/cmd/deepcopy-gen
GO111MODULE=off go get -u github.com/golang/protobuf/protoc-gen-go
GO111MODULE=off go get -u golang.org/x/tools/cmd/stringer

