GOPATH?=~/go
GOBIN?=${GOPATH}/bin

PROTOS = $(wildcard pkg/*/apis/**/*.proto)
PBS = $(PROTOS:%.proto=%.pb)

all: protos deviceplugin
protos: ${PBS}

%.pb: %.proto
	PATH=~/bin:$${PATH}:$(GOBIN) protoc -I $(dir $*.proto) $*.proto --go_out=plugins=grpc:$(dir $*.proto)

deviceplugin:
	@echo Building deviceplugin
	cd deviceplugin && go build

.PHONY: deviceplugin all
