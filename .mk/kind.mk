# Copyright 2019 VMware, Inc.
# SPDX-License-Identifier: Apache-2.0
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at:
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

KIND_CLUSTER_NAME="nsm"
KIND_IMAGE_PATH=$(IMAGE_DIR)

.PHONY: kind-config
kind-config:
	@which kind >/dev/null 2>&1 || \
		make kind-install

.PHONY: kind-install
kind-install:
	GO111MODULE="on" go get -u sigs.k8s.io/kind@master

.PHONY: kind-start
kind-start: kind-config
	@kind get clusters | grep nsm  >/dev/null 2>&1 && exit 0 || \
		( kind create cluster --name="$(KIND_CLUSTER_NAME)" --config ./scripts/kind.yaml --wait 120s && \
		until \
			KUBECONFIG="$$(kind get kubeconfig-path --name="$(KIND_CLUSTER_NAME)")" \
			kubectl taint node $(KIND_CLUSTER_NAME)-control-plane node-role.kubernetes.io/master:NoSchedule- ; \
		do echo "Waiting for the cluster to come up" && sleep 3; done )

.PHONY: kind-config-location
kind-config-location:
	@kind get kubeconfig-path --name="$(KIND_CLUSTER_NAME)"

.PHONY: kind-stop
kind-stop:
	@kind delete cluster --name="$(KIND_CLUSTER_NAME)"

.PHONY: kind-restart
kind-restart: kind-stop kind-start
	@echo "kind cluster restarted"

.PHONY: kind-%-load-images
kind-%-load-images:
	@if [ -e "$(KIND_IMAGE_PATH)/$*.tar" ]; then \
		echo "Loading image $*.tar to kind"; \
		kind load image-archive --name="$(KIND_CLUSTER_NAME)" $(KIND_IMAGE_PATH)$*.tar ; \
	else \
		echo "Cannot load $*.tar: $(IMAGE_DIR)/$*.tar does not exist.  Try running 'make k8s-$*-save'"; \
		exit 1; \
	fi
