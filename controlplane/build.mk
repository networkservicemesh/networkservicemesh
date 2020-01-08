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

controlplane_apps = $(shell ls ./controlplane/cmd/)
controlplane_targets = $(addsuffix -build, $(addprefix go-, $(controlplane_apps)))

.PHONY: $(controlplane_targets)
$(controlplane_targets): go-%-build: controlplane-%-build

.PHONY: docker-controlplane-list
docker-controlplane-list:
	@echo $(controlplane_apps)

.PHONY: docker-controlplane-build
docker-controlplane-build: $(addsuffix -build, $(addprefix docker-, $(controlplane_apps)))

.PHONY: docker-controlplane-save
docker-controlplane-save: $(addsuffix -save, $(addprefix docker-, $(controlplane_apps)))

.PHONY: docker-controlplane-push
docker-controlplane-push: $(addsuffix -push, $(addprefix docker-, $(controlplane_apps)))
