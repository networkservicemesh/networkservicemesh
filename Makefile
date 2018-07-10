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

all: check verify docker-build

check:
	@shellcheck `find . -name "*.sh" -not -path "./vendor/*"`

verify:
	@./scripts/verify-codegen.sh

docker-build:
	@docker build -t ligato/networkservicemesh/netmesh-test -f build/nsm/docker/Test.Dockerfile .
	@docker build -t ligato/networkservicemesh/netmesh -f build/nsm/docker/Dockerfile .
	@docker build -t ligato/networkservicemesh/nsm-init -f build/nsm-init/docker/Dockerfile .

