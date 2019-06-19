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

.PHONY: docker-build-arm64
docker-build-arm64: $(addsuffix -build-arm64,$(addprefix docker-,$(BUILD_CONTAINERS)))

.PHONY: docker-%-build-arm64
docker-%-build-arm64: install-qemu
	@${DOCKERBUILD} --network="host" --build-arg VPP_AGENT=${VPP_AGENT} -t ${ORG}/$*-arm64 -f docker/Dockerfile.$*-arm64 . && \
	if [ "x${COMMIT}" != "x" ] ; then \
		docker tag ${ORG}/$*-arm64 ${ORG}/$*-arm64:${COMMIT} ;\
	fi

.PHONY: docker-save-arm64
docker-save-arm64: $(addsuffix -save-arm64,$(addprefix docker-,$(BUILD_CONTAINERS)))

.PHONY: docker-%-save-arm64
docker-%-save-arm64: docker-%-build-arm64
	@echo "Saving $*"
	@mkdir -p scripts/vagrant/images/
	@docker save -o scripts/vagrant/images/$*-arm64.tar ${ORG}/$*-arm64

.PHONY: docker-run-arm64
docker-run-arm64: $(addsuffix -run-arm64,$(addprefix docker-,$(RUN_CONTAINERS)))

.PHONY: docker-%-run-arm64
docker-%-run-arm64: docker-%-build-arm64 docker-%-kill-arm64
	@echo "Starting $*-arm64..."
	@docker run -d --privileged=true -v "/var/lib/networkservicemesh:/var/lib/networkservicemesh" networkservicemesh/$*-arm64 > /tmp/container.$*-arm64

PHONY: docker-kill-arm64
docker-kill-arm64: $(addsuffix -kill-arm64,$(addprefix docker-,$(KILL_CONTAINERS)))

.PHONY: docker-%-kill-arm64
docker-%-kill-arm64:
	@echo "Killing $*-arm64... $$(cat /tmp/container.$*-arm64 | cut -c1-12)"
	@docker container ls | grep $$(cat /tmp/container.$*-arm64 | cut -c1-12) > /dev/null && xargs docker kill < /tmp/container.$*-arm64 || echo "$*-arm64 already killed"


.PHONY: docker-logs-arm64
docker-logs-arm64: $(addsuffix -logs-arm64,$(addprefix docker-,$(LOG_CONTAINERS)))

.PHONY: docker-%-logs-arm64
docker-%-logs-arm64:
	@echo "Showing nsmd logs..."
	@xargs docker logs < /tmp/container.$*-arm64

.PHONY:
docker-devenv-build-arm64: docker/debug/Dockerfile.debug-arm64
	@${DOCKERBUILD} -t networkservicemesh/devenv -f docker/debug/Dockerfile.debug-arm64 .

.PHONY: docker-%-push-arm64
docker-%-push-arm64: docker-login docker-%-build-arm64
	docker tag ${ORG}/$*-arm64:${COMMIT} ${ORG}/$*-arm64:${TAG}
	docker push ${ORG}/$*-arm64:${TAG}

PHONY: docker-push-arm64
docker-push-arm64: $(addsuffix -push-arm64,$(addprefix docker-,$(BUILD_CONTAINERS)))
