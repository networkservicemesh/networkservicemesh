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

.PHONY: skydive-url
skydive-url:
	@echo "$$($(kubectl) cluster-info | grep master | awk '{print $$6}' | sed 's/https/http/' | awk -F: '{print $$1":"$$2}'| sed $$'s,\x1b\\[[0-9;]*[a-zA-Z],,g'):$$($(kubectl) get svc skydive-analyzer | grep -v "NAME" | awk '{print $$5}' | awk -F, '{print $$1}' | awk -F/ '{print $$1}' | awk -F: '{print $$2}')"

.PHONY: skydive-port-forward
skydive-port-forward:
	@$(kubectl) port-forward svc/skydive-analyzer 8082:8082
