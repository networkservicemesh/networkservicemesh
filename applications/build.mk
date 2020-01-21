# Copyright (c) 2020 Cisco and/or its affiliates.
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

app_images = $(shell ls -d ./applications/*/ | xargs -n 1 basename)
apps_targets = $(addsuffix -build, $(addprefix go-, $(app_images)))

.PHONY: $(apps_targets)
$(apps_targets): go-%-build:
	@echo "----------------------  Building applications::$* via Cross compile ----------------------" && \
	pushd ./applications/$* && \
	${GO_BUILD} -o $(BIN_DIR)/$*/$* ./cmd/main.go && \
	popd

.PHONY: docker-applications-build
docker-applications-build: $(addsuffix -build, $(addprefix docker-, $(app_images)))

.PHONY: docker-applications-save
docker-applications-save: $(addsuffix -save, $(addprefix docker-, $(app_images)))

.PHONY: docker-applications-push
docker-applications-push: $(addsuffix -push, $(addprefix docker-, $(app_images)))

.PHONY: docker-applications-list
docker-applications-list:
	@echo $(app_images)
