GOPATH?=~/go
GOBIN?=${GOPATH}/bin

PROTOS = $(wildcard networkservice/*.proto)
PBS = $(PROTOS:%.proto=%.pb)

all: protos
protos: ${PBS}

%.pb: %.proto
	PATH=$(PATH):$(GOBIN) protoc -I networkservice $*.proto --go_out=plugins=grpc:go
