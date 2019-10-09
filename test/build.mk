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

test_apps = $(shell ls ./test/applications/cmd/)
test_targets = $(addsuffix -build, $(addprefix go-, $(test_apps)))

# TODO: files in test doesn't follow the regular structure: ./module/cmd/app,
# we should get rid of 'application' directory to have for example ./test/cmd/icmp-responder-nse
.PHONY: $(test_targets)
$(test_targets): go-%-build:
	@echo "----------------------  Building test/applications::$* via Cross compile ----------------------" && \
	pushd ./test && \
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) go build \
    	-ldflags "-extldflags '-static' -X  main.version=$(VERSION)" -o $(BIN_DIR)/$*/$* ./applications/cmd/$* && \
	popd

#TODO: get rid of 'common' images
test_common_dep = icmp-responder-nse monitoring-nsc monitoring-dns-nsc spire-proxy
docker-test-common-prepare: docker-%-prepare: $(addsuffix -build, $(addprefix go-, $(test_common_dep)))
	$(info Preparing files for docker...)
	$(call docker_prepare, $(BIN_DIR)/$*, \
			$(foreach app, $(test_common_dep), $(BIN_DIR)/$(app)/$(app)))

vpp_test_common_dep = vppagent-nsc vppagent-icmp-responder-nse vppagent-firewall-nse
docker-vpp-test-common-prepare: docker-%-prepare: $(addsuffix -build, $(addprefix go-, $(vpp_test_common_dep)))
	$(info Preparing files for docker...)
	$(call docker_prepare, $(BIN_DIR)/$*, \
		$(foreach app, $(vpp_test_common_dep), $(BIN_DIR)/$(app)/$(app)) \
		dataplane/vppagent/conf/vpp/startup.conf \
		test/applications/vpp-conf/supervisord.conf \
		test/applications/vpp-conf/run.sh)

.PHONY: docker-test-build
docker-test-build: docker-test-common-build docker-vpp-test-common-build

.PHONY: docker-test-save
docker-test-save: docker-test-common-save docker-vpp-test-common-save

.PHONY: docker-test-push
docker-test-push: docker-test-common-build docker-vpp-test-common-build
	docker tag ${ORG}/$*:${COMMIT} ${ORG}/test-common:${TAG}
	docker push ${ORG}/test-common:${TAG}
	docker tag ${ORG}/$*:${COMMIT} ${ORG}/vpp-test-common:${TAG}
    docker push ${ORG}/vpp-test-common:${TAG}
