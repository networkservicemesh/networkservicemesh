#!/bin/bash
version=v1.21.0

function get_linter() {
    echo "Installing golangci-lint: ${1}"
    wget -O - -q ${1} | sh -s -- ${version}
    sudo cp ./bin/golangci-lint "$(go env GOPATH)"/bin/
    sudo rm -rf ./bin/golangci-lint
}

{
  get_linter https://install.goreleaser.com/github.com/golangci/golangci-lint.sh
} || {
  get_linter https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh
} || {
  GO111MODULE=on go get github.com/golangci/golangci-lint/cmd/golangci-lint@${version}
}
