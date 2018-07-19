# Prerequisites to build

You will need to install

1. golang
2. protobuf
3. dep
4. protoc-gen-go

## On a Mac:

```bash
brew install dep golang jq protobuf shellcheck
go get -u github.com/golang/protobuf/protoc-gen-go
```

# Building

```bash
go generate ./...
go build ./...
go test ./...
```

# Regenerating Generated Code

To generate the deepcopy functions, clientset, listers and informers, run the following command:

```
GOPATH=<path to go base> ./scripts/update-codegen.sh
```

To regenerate the deepcopy code need for our CRD code, run the following commands:

```
$GOPATH/bin/deepcopy-gen --input-dirs ./netmesh/model/netmesh --go-header-file conf/boilerplate.txt --bounding-dirs ./netmesh/model/netmesh -O zz_generated.deepcopy -o $GOPATH/src
```

# Canonical source on how to build

The travis.yml file in the project provides the canonical source on how to
build in case this file is not properly maintained.
