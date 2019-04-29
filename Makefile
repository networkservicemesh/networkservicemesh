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

# Default target, no other targets should be before default
.PHONY: default
default: all

# Pull in k8s targets
include .mk/k8s.mk
include .mk/skydive.mk
include .mk/jaeger.mk
include .mk/monitor.mk
include .mk/integration.mk

GOPATH?=$(shell go env GOPATH 2>/dev/null)
GOCMD=go
GOFMT=${GOCMD} fmt
GOGET=${GOCMD} get
GOGENERATE=${GOCMD} generate
GOINSTALL=${GOCMD} install
GOTEST=${GOCMD} test
GOVET=${GOCMD} vet --all

# Export some of the above variables so they persist for the shell scripts
# which are run from the Makefiles
export GOPATH \
       GOCMD \
       GOFMT \
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
#
# The all target is what is used by the travis-ci system to build the Docker images
# which are used to run the code in each run.
#
all: check verify docker-build

check:
	@shellcheck `find . -name "*.sh" -not -path "*vendor/*"`

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
	@${GOVET} ./...

# Travis
.PHONY: travis
travis:
	@echo "=> TRAVIS: $$TRAVIS_BUILD_STAGE_NAME"
	@echo "Build: #$$TRAVIS_BUILD_NUMBER ($$TRAVIS_BUILD_ID)"
	@echo "Job: #$$TRAVIS_JOB_NUMBER ($$TRAVIS_JOB_ID)"
	@echo "AllowFailure: $$TRAVIS_ALLOW_FAILURE TestResult: $$TRAVIS_TEST_RESULT"
	@echo "Type: $$TRAVIS_EVENT_TYPE PullRequest: $$TRAVIS_PULL_REQUEST"
	@echo "Repo: $$TRAVIS_REPO_SLUG Branch: $$TRAVIS_BRANCH"
	@echo "Commit: $$TRAVIS_COMMIT"
	@echo "$$TRAVIS_COMMIT_MESSAGE"
	@echo "Range: $$TRAVIS_COMMIT_RANGE"
	@echo "Files:"
	@echo "$$(git diff --name-only $$TRAVIS_COMMIT_RANGE)"

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
