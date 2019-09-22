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

BUILD_CONTAINERS=nsmd nsmdp nsmd-k8s
BUILD_CONTAINERS+=proxy-nsmd proxy-nsmd-k8s
BUILD_CONTAINERS+=crossconnect-monitor
BUILD_CONTAINERS+=nsm-init nsm-monitor nsm-coredns
BUILD_CONTAINERS+=test-common
BUILD_CONTAINERS+=nsm-coredns
BUILD_CONTAINERS+=admission-webhook
BUILD_CONTAINERS+=proxy-nsmgr

# Set the configured forwarding plane
ifeq (${FORWARDING_PLANE}, vpp)
  BUILD_CONTAINERS+=vppagent-dataplane vpp-test-common
else ifeq (${FORWARDING_PLANE}, kernel)
  BUILD_CONTAINERS+=kernel-forwarder
endif


RUN_CONTAINERS=$(BUILD_CONTAINERS)
KILL_CONTAINERS=$(BUILD_CONTAINERS)
LOG_CONTAINERS=$(KILL_CONTAINERS)
ORG=networkservicemesh
GO_VERSION=1.13
GOPROXY=https://proxy.golang.org
VERSION = $(shell git describe --tags --always)

include .mk/vpp_agent.mk

.PHONY: docker-build
docker-build: $(addsuffix -build,$(addprefix docker-,$(BUILD_CONTAINERS)))

.PHONY: docker-%-build
docker-%-build::
	@@./build/build.sh $* ${ORG}/$* ; \
	if [ "x${COMMIT}" != "x" ] ; then \
		docker tag ${ORG}/$* ${ORG}/$*:${COMMIT} ;\
	fi

.PHONY: docker-test-common-build
docker-test-common-build:
	@./build/build-test-common.sh test-common ${ORG}/test-common; \
	if [ "x${COMMIT}" != "x" ] ; then \
    		docker tag ${ORG}/test-common ${ORG}/test-common:${COMMIT} ;\
	fi

.PHONY: docker-vpp-test-common-build
docker-vpp-test-common-build:
	@./build/build-vpp-test-common.sh vpp-test-common ${ORG}/vpp-test-common; \
	if [ "x${COMMIT}" != "x" ] ; then \
    		docker tag ${ORG}/vpp-test-common ${ORG}/vpp-test-common:${COMMIT} ;\
	fi

.PHONY: docker-vppagent-dataplane-build
docker-vppagent-dataplane-build:
	@./build/build-vppagent-dataplane.sh vppagent-dataplane ${ORG}/vppagent-dataplane
	if [ "x${COMMIT}" != "x" ] ; then \
    		docker tag ${ORG}/vppagent-dataplane ${ORG}/vppagent-dataplane:${COMMIT} ;\
	fi

.PHONY: docker-nsm-coredns-build
docker-nsm-coredns-build:
	@./build/build-nsm-coredns.sh nsm-coredns ${ORG}/nsm-coredns; \
	if [ "x${COMMIT}" != "x" ] ; then \
    		docker tag ${ORG}/nsm-coredns ${ORG}/nsm-coredns:${COMMIT} ;\
	fi


.PHONY: docker-save
docker-save: $(addsuffix -save,$(addprefix docker-,$(BUILD_CONTAINERS)))

.PHONY: docker-%-save
docker-%-save: docker-%-build
	@echo "Saving $* to scripts/vagrant/images/$*.tar"
	@mkdir -p scripts/vagrant/images/
	@docker save -o scripts/vagrant/images/$*.tar ${ORG}/$*

.PHONY: docker-run
docker-run: $(addsuffix -run,$(addprefix docker-,$(RUN_CONTAINERS)))

.PHONY: docker-%-run
docker-%-run: docker-%-build docker-%-kill
	@echo "Starting $*..."
	@docker run -d --privileged=true -v "/var/lib/networkservicemesh:/var/lib/networkservicemesh" networkservicemesh/$* > /tmp/container.$*

PHONY: docker-kill
docker-kill: $(addsuffix -kill,$(addprefix docker-,$(KILL_CONTAINERS)))

.PHONY: docker-%-kill
docker-%-kill:
	@echo "Killing $*... $$(cat /tmp/container.$* | cut -c1-12)"
	@docker container ls | grep $$(cat /tmp/container.$* | cut -c1-12) > /dev/null && xargs docker kill < /tmp/container.$* || echo "$* already killed"

.PHONY: docker-logs
docker-logs: $(addsuffix -logs,$(addprefix docker-,$(LOG_CONTAINERS)))

.PHONY: docker-%-logs
docker-%-logs:
	@echo "Showing nsmd logs..."
	@xargs docker logs < /tmp/container.$*

.PHONY:
docker-devenv-build: docker/debug/Dockerfile.debug
	@${DOCKERBUILD} --build-arg VENDORING="${VENDORING}" -t networkservicemesh/devenv -f docker/debug/Dockerfile.debug .

.PHONY: docker-devenv-run
docker-devenv-run:
	@docker run -d --privileged -p 40000-40100:40000-40100/tcp -v "/var/lib/networkservicemesh:/var/lib/networkservicemesh"  -v $$(pwd | rev | cut -d'/' -f4- | rev):/go/src networkservicemesh/devenv > /tmp/container.devenv
	@xargs docker logs -f < /tmp/container.devenv

.PHONY: docker-devenv-kill
docker-devenv-kill:
	@docker kill $$(docker ps | grep networkservicemesh/devenv | cut -c1-12) 2&>1 > /dev/null || echo "DevEnv already killed"

.PHONY: docker-devenv-attach
docker-devenv-attach:
	@docker exec -ti $$(docker container ls | grep networkservicemesh/devenv | cut -c1-12) bash


.PHONY: docker-%-debug
docker-%-debug:
	@docker exec -ti $$(docker container ls | grep networkservicemesh/devenv | cut -c1-12) /go/src/github.com/networkservicemesh/networkservicemesh/scripts/debug.sh $*

.PHONY: docker-%-push
docker-%-push: docker-login docker-%-build
	docker tag ${ORG}/$*:${COMMIT} ${ORG}/$*:${TAG}
	docker push ${ORG}/$*:${TAG}

PHONY: docker-push
docker-push: $(addsuffix -push,$(addprefix docker-,$(BUILD_CONTAINERS)))

.PHONY: docker-login
docker-login:
	@echo "${DOCKER_PASSWORD}" | docker login -u "${DOCKER_USERNAME}" --password-stdin




