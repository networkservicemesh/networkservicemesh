GOPATH?=~/go
GOBIN?=${GOPATH}/bin

PROTOS = $(wildcard pkg/*/apis/**/*.proto)
PBS = $(PROTOS:%.proto=%.pb)

all: protos
protos: ${PBS}

%.pb: %.proto
	PATH=~/bin:$${PATH}:$(GOBIN) protoc -I $(dir $*.proto) $*.proto --go_out=plugins=grpc:$(dir $*.proto)
