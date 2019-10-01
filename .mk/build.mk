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

modules := controlplane k8s side-cars test dataplane

include $(foreach module, $(modules), ./$(module)/build.mk)

BIN_DIR = $(PWD)/build/dist
VERSION = $(shell git describe --tags --always)
VPP_AGENT=ligato/vpp-agent:v2.1.1

define docker_prepare
    @mkdir -p $1; \
    for app in $2; do \
        cp $$app $1; \
    done
endef

.PHONY: docker-%-build
docker-%-build: go-%-build
	./scripts/build_image.sh -o ${ORG} -a $* -b $(BIN_DIR)/$* -e $*

.PHONY: docker-build
docker-build: $(addsuffix -build, $(addprefix docker-, $(modules)))

.PHONY: docker-%-save
docker-%-save: docker-%-build
	@echo "Saving $* to scripts/vagrant/images/$*.tar"
	@mkdir -p scripts/vagrant/images/
	@docker save -o scripts/vagrant/images/$*.tar ${ORG}/$*

.PHONY: docker-save
docker-save: $(addsuffix -save, $(addprefix docker-, $(modules)))

.PHONY: docker-%-push
docker-%-push: docker-login docker-%-build
	docker tag ${ORG}/$*:${COMMIT} ${ORG}/$*:${TAG}
	docker push ${ORG}/$*:${TAG}

PHONY: docker-push
docker-push: $(addsuffix -push,$(addprefix docker-,$(BUILD_CONTAINERS)))

.PHONY: docker-login
docker-login:
	@echo "${DOCKER_PASSWORD}" | docker login -u "${DOCKER_USERNAME}" --password-stdin

clean:
	rm -rf $(BIN_DIR)
