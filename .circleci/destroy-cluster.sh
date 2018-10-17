#!/bin/bash
#
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

K8S_DEPLOYMENT_NAME=nsm-ci-"${CIRCLE_WORKFLOW_ID:0:8}"

# Circle CI use Remote Docker and do not support folder mount.
# So we create a dummy container which will hold a volume with deplyment data
docker create -v /cncf/data --name configs alpine:3.7 /bin/true
# copy a deployment data into this volume
docker cp data/. configs:/cncf/data
# start cross-cloud container using this volume
docker run \
  --volumes-from configs \
  --dns 147.75.69.23 --dns 8.8.8.8 \
  -e NAME="${K8S_DEPLOYMENT_NAME}" \
  -e CLOUD=packet \
  -e COMMAND=destroy \
  -e BACKEND=file \
  -e PACKET_AUTH_TOKEN="${PACKET_AUTH_TOKEN}" \
  -e TF_VAR_packet_project_id="${PACKET_PROJECT_ID}" \
  -ti registry.cncf.ci/cncf/cross-cloud/provisioning:production
