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

# TODO: files in forwarder doesn't follow the regular structure: ./module/cmd/app,
# after fixing 'kernel-forwarder' and 'vppagent-forwarder' targets could be eliminated
.PHONY: go-kernel-forwarder-build
go-kernel-forwarder-build: go-%-build:
	$(info ----------------------  Building forwarder::$* via Cross compile ----------------------)
	@pushd ./forwarder && \
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) go build \
    	-ldflags "-extldflags '-static' -X  main.version=$(VERSION)" -o $(BIN_DIR)/$*/$* ./kernel-forwarder/cmd/ && \
	popd

.PHONY: go-vppagent-forwarder-build
go-vppagent-forwarder-build: go-%-build:
	$(info ----------------------  Building forwarder::$* via Cross compile ----------------------)
	@pushd ./forwarder && \
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) go build \
    	-ldflags "-extldflags '-static' -X  main.version=$(VERSION)" -o $(BIN_DIR)/$*/$* ./vppagent/cmd/ && \
	popd

docker-vppagent-forwarder-prepare: docker-%-prepare: go-%-build
	$(info Preparing files for docker...)
	$(call docker_prepare, $(BIN_DIR)/$*, \
		forwarder/vppagent/conf/vpp/startup.conf \
		forwarder/vppagent/conf/supervisord/supervisord.conf \
		forwarder/vppagent/conf/supervisord/govpp.conf)

.PHONY: docker-forwarder-build
docker-forwarder-build: docker-vppagent-forwarder-build docker-kernel-forwarder-build

.PHONY: docker-forwarder-save
docker-forwarder-save: docker-vppagent-forwarder-save docker-kernel-forwarder-save

.PHONY: docker-forwarder-push
docker-forwarder-push: docker-vppagent-forwarder-push docker-kernel-forwarder-push

