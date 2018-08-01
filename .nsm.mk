# Copyright (c) 2018 Cisco and/or its affiliates.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This file includes definitions for Docker images used by the Makefile
# and docker build infrastructure. It also contains the targets to build
# and push Docker images

DOCKER_NETMESH_TEST=networkservicemesh/netmesh-test
DOCKER_NETMESH=networkservicemesh/netmesh
DOCKER_SIMPLE_DATAPLANE=networkservicemesh/simple-dataplane
DOCKER_NSM_INIT=networkservicemesh/nsm-init
DOCKER_NSE=networkservicemesh/nse
DOCKER_RELEASE=networkservicemesh/release

#
# Targets to build docker images
#
.PHONY: docker-build-netmesh-test
docker-build-netmesh-test:
	@${DOCKERBUILD} -t ${DOCKER_NETMESH_TEST} -f build/nsm/docker/Test.Dockerfile .

.PHONY: docker-build-release
docker-build-release:
	@${DOCKERBUILD} -t ${DOCKER_RELEASE} -f build/Dockerfile .

.PHONY: docker-build-netmesh
docker-build-netmesh: docker-build-release
	@${DOCKERBUILD} -t ${DOCKER_NETMESH} -f build/nsm/docker/Dockerfile .

.PHONY: docker-build-simple-dataplane
docker-build-simple-dataplane: docker-build-release
	@${DOCKERBUILD} -t ${DOCKER_SIMPLE_DATAPLANE} -f build/simple-dataplane/docker/Dockerfile .

.PHONY: docker-build-nsm-init
docker-build-nsm-init: docker-build-release
	@${DOCKERBUILD} -t ${DOCKER_NSM_INIT} -f build/nsm-init/docker/Dockerfile .

.PHONY: docker-build-nse
docker-build-nse: docker-build-release
	@${DOCKERBUILD} -t ${DOCKER_NSE} -f build/nse/docker/Dockerfile .

