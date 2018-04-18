GOPATH?=~/go
GOBIN?=${GOPATH}/bin

PROTOS = $(wildcard networkservice/*.proto)
PBS = $(PROTOS:%.proto=%.pb)

all: protos
protos: ${PBS}

%.pb: %.proto
	PATH=$(PATH):$(GOBIN):~/bin protoc -I networkservice $*.proto --go_out=plugins=grpc:go
