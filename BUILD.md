# Prerequisites to build

You will need to install

1. golang
2. protobuf
3. dep
4. protoc-gen-go

## On a Mac:

```bash
brew install golang protobuf dep
go get -u github.com/golang/protobuf/protoc-gen-go
```

# Building

```bash
go generate ./...
go build ./...
go test ./...
```

# Canonical source on how to build

The travis.yml file in the project provides the canonical source on how to
build in case this file is not properly maintained.
