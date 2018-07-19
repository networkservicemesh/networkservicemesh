# Copyright (c) 2018 Cisco and/or its affiliates.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
#  You may obtain a copy of the License at:
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

GOPATH?=$(shell go env GOPATH)
GOCMD=go
GOFMT=${GOCMD} fmt
GOGET=${GOCMD} get
GOGENERATE=${GOCMD} generate
GOINSTALL=${GOCMD} install
GOTEST=${GOCMD} test
GOVET=${GOCMD} tool vet
GOVETTARGETS=cmd \
	pkg/apis/networkservicemesh.io/v1 \
	pkg/nsm \
	plugins \
	utils

# Setup proxies for docker build
ifeq ($(HTTP_PROXY),)
HTTPBUILD=
else
HTTPBUILD=--build-arg HTTP_PROXY=$(HTTP_PROXY)
endif
ifeq ($(HTTPS_PROXY),)
HTTPSBUILD=
else
HTTPSBUILD=--build-arg HTTPS_PROXY=$(HTTPS_PROXY)
endif

DOCKERBUILD=docker build ${HTTPBUILD} ${HTTPSBUILD}

.PHONY: all check verify docker-build
#
# The all target is what is used by the travis-ci system to build the Docker images
# which are used to run the code in each run.
#
all: check verify docker-build

check:
	@shellcheck `find . -name "*.sh" -not -path "./vendor/*"`

verify:
	@./scripts/verify-codegen.sh

docker-build: docker-build-netmesh-test docker-build-netmesh docker-build-nsm-init docker-build-nse

.PHONY: docker-build-netmesh-test
docker-build-netmesh-test:
	${DOCKERBUILD} -t ligato/networkservicemesh/netmesh-test -f build/nsm/docker/Test.Dockerfile .

.PHONY: docker-build-netmesh
docker-build-netmesh:
	${DOCKERBUILD} -t ligato/networkservicemesh/netmesh -f build/nsm/docker/Dockerfile .

.PHONY: docker-build-simple-dataplane
docker-build-simple-dataplane:
	@docker build -t ligato/networkservicemesh/simple-dataplane -f build/simple-dataplane/docker/Dockerfile .

.PHONY: docker-build-nsm-init
docker-build-nsm-init:
	${DOCKERBUILD} -t ligato/networkservicemesh/nsm-init -f build/nsm-init/docker/Dockerfile .

.PHONY: docker-build-nse
docker-build-nse:
	${DOCKERBUILD} -t ligato/networkservicemesh/nse -f build/nse/docker/Dockerfile .

.PHONY: format deps generate install test test-race vet
#
# The following targets are meant to be run when working with the code locally.
#
format:
	@${GOFMT} ./...

deps:
	@${GOGET} -u github.com/golang/protobuf/protoc-gen-go

generate:
	@${GOGENERATE} ./...

install:
	@${GOINSTALL} ./...

test:
	@${GOTEST} ./... -cover

test-race:
	@${GOTEST} -race ./... -cover

vet:
	${GOVET} ${GOVETTARGETS}

# Test target to debug proxy issues
checkproxy:
	echo "HTTPBUILD=${HTTPBUILD} HTTPSBUILD=${HTTPSBUILD}"
	echo "DOCKERBUILD=${DOCKERBUILD}"
