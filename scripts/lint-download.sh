#!/bin/bash

{
    wget -O - -q https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s -- v1.17.1
    sudo cp ./bin/golangci-lint "$(go env GOPATH)"/bin/
} || {
    wget -O - -q https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- v1.17.1
    sudo cp ./bin/golangci-lint "$(go env GOPATH)"/bin/
} || {
    make lint-install
}
