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

.PHONY: $(test_targets)
$(test_targets): go-%-build:
	./scripts/build.sh $* ./test/applications ./cmd/$* $(BIN_DIR)/$* $(VERSION)

test_common_dep = icmp-responder-nse monitoring-nsc monitoring-dns-nsc spire-proxy
docker-test-common-build: docker-%-build: $(addsuffix -build, $(addprefix go-, $(test_common_dep)))
	$(call docker_prepare, $(BIN_DIR)/$*, \
			$(foreach app, $(test_common_dep), $(BIN_DIR)/$(app)/$(app)))
	./scripts/build_image.sh -o ${ORG} -a $* -b $(BIN_DIR)/$*

vpp_test_common_dep = vppagent-nsc vppagent-icmp-responder-nse vppagent-firewall-nse
docker-vpp-test-common-build: docker-%-build: $(addsuffix -build, $(addprefix go-, $(vpp_test_common_dep)))
	$(call docker_prepare, $(BIN_DIR)/$*, \
		$(foreach app, $(vpp_test_common_dep), $(BIN_DIR)/$(app)/$(app)) \
		dataplane/vppagent/conf/vpp/startup.conf \
		test/applications/vpp-conf/supervisord.conf \
		test/applications/vpp-conf/run.sh)
	./scripts/build_image.sh -o ${ORG} -a $* -b $(BIN_DIR)/$* -g VPP_AGENT=$(VPP_AGENT)

.PHONY: docker-test-build
docker-test-build: $(addsuffix -build, $(addprefix docker-, $(test_apps) test-common vpp-test-common))

.PHONY: docker-test-save
docker-test-save: $(addsuffix -save, $(addprefix docker-, $(test_apps) test-common vpp-test-common))
