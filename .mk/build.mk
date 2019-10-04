# Copyright (c) 2019 Cisco and/or its affiliates.
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

modules = $(filter-out mk,$(subst .,,$(subst /,,$(dir $(shell find . -name build.mk)))))

include $(foreach module, $(modules), ./$(module)/build.mk)

BIN_DIR = $(PWD)/build/dist
VERSION = $(shell git describe --tags --always)
VPP_AGENT=ligato/vpp-agent:v2.3.0
CGO_ENABLED=0
GOOS=linux
DOCKER=./build

print:
	echo $(modules)

define docker_prepare
	@mkdir -p $1; \
	for app in $2; do \
		cp $$app $1; \
	done
endef

define build_rule
$(module)-%-build:
	@echo "----------------------  Building ${module}::$$* via Cross compile ----------------------" && \
	pushd ./$(module) && \
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) go build \
    	-ldflags "-extldflags '-static' -X  main.version=$(VERSION)" -o $(BIN_DIR)/$$*/$$* ./cmd/$$* && \
	popd
endef

define docker_build
	docker build --build-arg VPP_AGENT=$(VPP_AGENT) --build-arg ENTRY=$1 --network="host" -t $(ORG)/$1 -f $(DOCKER)/$2 $3; \
	if [ "x${CONTAINER_TAG}" != "x" ] ; then \
		docker tag $(ORG)/$1 $(ORG)/$1:${CONTAINER_TAG} ;\
	fi
endef

$(foreach module, $(modules), $(eval $(call build_rule, $(module))))

images += $(modules)
images += spire-registration

.PHONY: docker-build
docker-build: $(addsuffix -build, $(addprefix docker-, $(images)))

# Builds docker image using $(BIN_DIR)/$* as Build Context
.PHONY: docker-%-build
docker-%-build: docker-%-prepare
	$(info Building docker image for $*)
	@if [ -f $(DOCKER)/Dockerfile.$* ]; then \
		$(call docker_build,$*,Dockerfile.$*, $(BIN_DIR)/$*); \
	else \
		$(call docker_build,$*,Dockerfile.empty,$(BIN_DIR)/$*); \
	fi

.PHONY: docker-spire-registration-build
docker-spire-registration-build: docker-%-build:
	$(call docker_build,$*,Dockerfile.$*,.)

# Could be overrided in ./module/build.mk files to copy some configs
# into $(BIN_DIR)/$* before sending it as a Build Context to docker
.PHONY: docker-%-prepare
docker-%-prepare: go-%-build
	$(info Nothing to prepare...)

.PHONY: docker-save
docker-save: $(addsuffix -save, $(addprefix docker-, $(images)))

.PHONY: docker-%-save
docker-%-save: docker-%-build
	@echo "Saving $* to $(IMAGE_DIR)/$*.tar"
	@mkdir -p $(IMAGE_DIR)
	@docker save -o $(IMAGE_DIR)/$*.tar ${ORG}/$*:$(CONTAINER_TAG)

.PHONY: docker-%-push
docker-%-push: docker-login docker-%-build
	docker push ${ORG}/$*:${CONTAINER_TAG}

.PHONY: docker-push
docker-push: $(addsuffix -push,$(addprefix docker-,$(images)));

.PHONY: docker-login
docker-login:
	@if [ "${AZURE_BUILD}" != 1 ] ; then \
		echo "${DOCKER_PASSWORD}" | docker login -u "${DOCKER_USERNAME}" --password-stdin; \
	fi

clean:
	rm -rf $(BIN_DIR)
