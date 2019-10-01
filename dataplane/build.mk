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

# TODO: files in dataplane doesn't follow the regular structure: ./module/cmd/app,
# after fixing 'kernel-forwarder' and 'vppagent-dataplane' targets could be eliminated
.PHONY: go-kernel-forwarder-build
go-kernel-forwarder-build: go-%-build:
	./scripts/build.sh $* dataplane ./$*/cmd $(BIN_DIR)/$* $(VERSION)

.PHONY: go-vppagent-dataplane-build
go-vppagent-dataplane-build: go-%-build:
	./scripts/build.sh $* dataplane ./vppagent/cmd $(BIN_DIR)/$* $(VERSION)

docker-vppagent-dataplane-build: docker-%-build: go-vppagent-dataplane-build
	$(call docker_prepare, $(BIN_DIR)/$*, \
		$(foreach app, $^, $(BIN_DIR)/$(app)/$(app)) \
		dataplane/vppagent/conf/vpp/startup.conf \
		dataplane/vppagent/conf/supervisord/supervisord.conf)
	./scripts/build_image.sh -o ${ORG} -a $* -b $(BIN_DIR)/$* -g VPP_AGENT=$(VPP_AGENT)

.PHONY: docker-dataplane-build
docker-dataplane-build: docker-vppagent-dataplane-build docker-kernel-forwarder-build

.PHONY: docker-dataplane-save
docker-dataplane-save: docker-vppagent-dataplane-save docker-kernel-forwarder-save