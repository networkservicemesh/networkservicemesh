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

BUILD_CONTAINERS=nsmd nsmdp nsmd-k8s icmp-responder-nse
RUN_CONTAINERS=$(BUILD_CONTAINERS)
KILL_CONTAINERS=$(BUILD_CONTAINERS)
LOG_CONTAINERS=$(KILL_CONTAINERS)


.PHONY: docker-build
docker-build: $(addsuffix -build,$(addprefix docker-,$(BUILD_CONTAINERS)))

.PHONY: docker-%-build
docker-%-build:
	@${DOCKERBUILD} -t networkservicemesh/$* -f docker/Dockerfile.$* .

.PHONY: docker-save
docker-save: $(addsuffix -save,$(addprefix docker-,$(BUILD_CONTAINERS)))

.PHONY: docker-%-save
docker-%-save: docker-%-build
	@echo "Saving $*"
	@mkdir -p ../scripts/vagrant/images/
	@docker save -o ../scripts/vagrant/images/$*.tar $*/$*

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
	@echo "Killing $*..."
	@docker container ls | grep $$(cat /tmp/container.$*| cut -c1-12) > /dev/null && xargs docker kill < /tmp/container.$* || echo "$* already killed"

.PHONY: docker-logs
docker-logs: $(addsuffix -logs,$(addprefix docker-,$(LOG_CONTAINERS)))

.PHONY: docker-%-logs
docker-%-logs:
	@echo "Showing nsmd logs..."
	@xargs docker logs < /tmp/container.$*

docker-build-debug: docker/debug/Dockerfile.debug
	@${DOCKERBUILD} -t networkservicemesh/debug -f docker/debug/Dockerfile.debug .

.PHONY: docker-debug-%
docker-debug-%: docker-build-debug docker-%-kill
	@${DOCKERBUILD} -t networkservicemesh/$*-debug -f build/Dockerfile.$*-debug .
	@docker run -d --privileged -p 127.0.0.1:40000:40000 -v "/var/lib/networkservicemesh:/var/lib/networkservicemesh" networkservicemesh/$*-debug > /tmp/container.$*
	@docker container ls | grep $$(cat /tmp/container.$*| cut -c1-12) > /dev/null && xargs docker logs < /tmp/container.$*

.PHONY: docker-push-%
docker-%-push: docker-login
	docker tag ${ORG}/$*:${COMMIT} ${ORG}/$*:${TAG}
	docker tag ${ORG}/$*:${COMMIT} ${ORG}/$*:${BUILD_TAG}
	docker push ${ORG}/$*




