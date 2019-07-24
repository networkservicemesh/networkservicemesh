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

# Default target, no other targets should be before default
.PHONY: default
default: all


GOPATH?=$(shell go env GOPATH)
GOCMD=go
GOIMPORTS=goimports
GOGET=${GOCMD} get
GOGENERATE=${GOCMD} generate
GOINSTALL=${GOCMD} install
GOTEST=${GOCMD} test

ORG=networkservicemesh
BUILD_CONTAINERS=test-common nsm-generate-sriov-configmap
RUN_CONTAINERS=dataplane $(BUILD_CONTAINERS)
KILL_CONTAINERS=$(BUILD_CONTAINERS)
LOG_CONTAINERS=$(KILL_CONTAINERS)

# Export some of the above variables so they persist for the shell scripts
# which are run from the Makefiles
export GOPATH \
       GOCMD \
       GOIMPORTS \
       GOGET \
       GOGENERATE \
       GOINSTALL \
       GOTEST

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

.PHONY: all docker-build

all: docker-build

docker-build: $(addsuffix -build,$(addprefix docker-,$(BUILD_CONTAINERS)))

.PHONY: docker-%-build
docker-%-build:
	@${DOCKERBUILD} -t networkservicemesh/$* -f build/Dockerfile.$* ..


docker-save: $(addsuffix -save,$(addprefix docker-,$(BUILD_CONTAINERS)))

#
# Targets to save docker images
#
.PHONY: docker-%-save
docker-%-save:
	@echo "Saving $*"
	@mkdir -p ../scripts/vagrant/images/
	@docker save -o ../scripts/vagrant/images/$*.tar $*/$*

.PHONY: docker-run
docker-run: $(addsuffix -run,$(addprefix docker-,$(RUN_CONTAINERS)))

.PHONY: docker-%-run
docker-%-run: docker-%-build docker-%-kill
	@echo "Starting $*..."
	@docker run -d -v "/var/lib/networkservicemesh:/var/lib/networkservicemesh" $*/$* > /tmp/container.$*

.PHONY: docker-dataplane-run
docker-dataplane-run:
	@echo "Starting vpp..."
	@docker run --network=host --privileged=true --volume=/var/run:/var/run --volume=/var/lib:/var/lib --volume=/lib/modules:/lib/modules --ipc=host --pid=host -d networkservicemesh/vpp > /tmp/container.vpp 
	@echo "Starting vpp-daemon..."
	@docker run --network=host --privileged=true --volume=/var/run:/var/run --volume=/var/lib:/var/lib --volume=/lib/modules:/lib/modules --volume=/var/lib/networkservicemesh:/var/lib/networkservicemesh/ --ipc=host --pid=host -d networkservicemesh/vpp-daemon > /tmp/container.vpp-daemon

.PHONY: docker-kill
docker-kill: $(addsuffix -kill,$(addprefix docker-,$(KILL_CONTAINERS)))

.PHONY: docker-%-kill
docker-%-kill:
	@echo "Killing $*..."
	@docker container ls | grep $$(cat /tmp/container.$*| cut -c1-12) > /dev/null && xargs docker kill < /tmp/container.$* || echo "$* already killed"

.PHONY: docker-logs
docker-logs: $(addsuffix -logs,$(addprefix docker-,$(LOG_CONTAINERS)))

.PHONY: docker-%-logs
docker-%-logs:
	@echo "Showing NSMgr logs..."
	@xargs docker logs < /tmp/container.$*


.PHONY: format deps generate install test test-race vet
#
# The following targets are meant to be run when working with the code locally.
#
format:
	@${GOIMPORTS} -w .

generate:
	@${GOGENERATE} ./...

install:
	@${GOINSTALL} ./...

test:
	@${GOTEST} ./... -cover

test-race:
	@${GOTEST} -race ./... -cover

# Test target to debug proxy issues
checkproxy:
	echo "HTTPBUILD=${HTTPBUILD} HTTPSBUILD=${HTTPSBUILD}"
	echo "DOCKERBUILD=${DOCKERBUILD}"
