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

sidecars_apps = $(shell ls ./side-cars/cmd/)
sidecars_targets = $(addsuffix -build, $(addprefix go-, $(sidecars_apps)))

.PHONY: docker-side-cars-list
docker-side-cars-list:
	@echo $(sidecars_apps)

.PHONY: $(sidecars_targets)
$(sidecars_targets): go-%-build: side-cars-%-build

.PHONY: docker-side-cars-build
docker-side-cars-build: $(addsuffix -build, $(addprefix docker-, $(sidecars_apps)))

.PHONY: docker-side-cars-save
docker-side-cars-save: $(addsuffix -save, $(addprefix docker-, $(sidecars_apps)))

.PHONY: docker-side-cars-push
docker-side-cars-push: $(addsuffix -push, $(addprefix docker-, $(sidecars_apps)))
