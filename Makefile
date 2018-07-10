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

GOCMD=go
GOFMT=${GOCMD} fmt
GOGET=${GOCMD} get
GOGENERATE=${GOCMD} generate
GOINSTALL=${GOCMD} install
GOTEST=${GOCMD} test

#
# The all target is what is used by the travis-ci system to build the Docker images
# which are used to run the code in each run.
#
all: check verify docker-build

check:
	@shellcheck `find . -name "*.sh" -not -path "./vendor/*"`

verify:
	@./scripts/verify-codegen.sh

docker-build:
	@docker build -t ligato/networkservicemesh/netmesh-test -f build/nsm/docker/Test.Dockerfile .
	@docker build -t ligato/networkservicemesh/netmesh -f build/nsm/docker/Dockerfile .
	@docker build -t ligato/networkservicemesh/nsm-init -f build/nsm-init/docker/Dockerfile .
	@docker build -t ligato/networkservicemesh/nse -f build/nse/docker/Dockerfile .

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
