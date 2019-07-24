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

# We want to use bash
SHELL:=/bin/bash
WORKER_COUNT ?= 1

# Set a default forwarding plane
FORWARDING_PLANE ?= vpp

# Default target, no other targets should be before default
.PHONY: default
default: all

# Static code analysis
include .mk/code_analysis.mk

# Pull in k8s targets
include .mk/k8s.mk
include .mk/skydive.mk
include .mk/jaeger.mk
include .mk/monitor.mk
include .mk/integration.mk

GOPATH?=$(shell go env GOPATH 2>/dev/null)
GOCMD=go
GOIMPORTS=goimports
GOGET=${GOCMD} get
GOGENERATE=${GOCMD} generate
GOINSTALL=${GOCMD} install
GOTEST=${GOCMD} test
GOVET=${GOCMD} vet --all

# Export some of the above variables so they persist for the shell scripts
# which are run from the Makefiles
export GOPATH \
       GOCMD \
       GOIMPORTS \
       GOGET \
       GOGENERATE \
       GOINSTALL \
       GOTEST \
       GOVET

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

.PHONY: all check verify # docker-build docker-push

all: check verify docker-build

check:
	@shellcheck `find . -name "*.sh" -not -path "*vendor/*"`

.PHONY: format deps generate install test test-race vet
#
# The following targets are meant to be run when working with the code locally.
#
format:
	@${GOIMPORTS} -w -local github.com/networkservicemesh/networkservicemesh -d $(find . -type f -name '*.go' -not -name '*.pb.go')

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
	@${GOVET} ./...

# Get dependency manager tool
get-dep:
	curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
	dep version

# Check state of dependencies
dep-check: get-dep
	@echo "=> checking dependencies"
	dep check

# Test target to debug proxy issues
checkproxy:
	echo "HTTPBUILD=${HTTPBUILD} HTTPSBUILD=${HTTPSBUILD}"
	echo "DOCKERBUILD=${DOCKERBUILD}"
