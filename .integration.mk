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

.PHONY: k8s-integration-config
k8s-integration-config:
	@kubectl label --overwrite --all=true nodes app=nsmd-ds
	@kubectl apply -f k8s/conf/cluster-role-admin.yaml
	@kubectl apply -f k8s/conf/cluster-role-binding.yaml
	@kubectl apply -f k8s/conf/crd-networkservices.yaml
	@kubectl apply -f k8s/conf/crd-networkserviceendpoints.yaml
	@kubectl apply -f k8s/conf/crd-networkservicemanagers.yaml

.PHONY: k8s-integration-tests
k8s-integration-tests: k8s-integration-config
	@GO111MODULE=on go test -v ./test/...

.PHONY: k8s-integration-%-test
k8s-integration-%-test: k8s-integration-config
	@GO111MODULE=on go test -v ./test/... -run $*