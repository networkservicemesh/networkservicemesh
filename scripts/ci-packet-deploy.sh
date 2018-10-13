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
#
# The purpose of the script is to deploy k8s cluster in the packet.net
# PACKET_AUTH_TOKEN and PACKET_PROJECT_ID MUST be provided externally

K8S_DEPLOYMENT_NAME=nsm-ci-"$CIRCLE_BUILD_NUM"

docker run \
  --name deploy-on-packet \
  --dns 147.75.69.23 --dns 8.8.8.8 \
  -e NAME="${K8S_DEPLOYMENT_NAME}" \
  -e CLOUD=packet    \
  -e COMMAND=deploy \
  -e BACKEND=file  \
  -e PACKET_AUTH_TOKEN="${PACKET_AUTH_TOKEN}" \
  -e TF_VAR_packet_project_id="${PACKET_PROJECT_ID}" \
  -ti registry.cncf.ci/cncf/cross-cloud/provisioning:production

docker cp deploy-on-packet:/cncf/data .data
mkdir "$HOME"/.kube
cp .data/kubeconfig "$HOME"/.kube/config

# Adding cross-cloud's nameserver to resolve cluster IP
cp /etc/resolv.conf resolv.conf
echo "echo \"nameserver 147.75.69.23\" > /etc/resolv.conf" | sudo sh
echo "cat resolv.conf >> /etc/resolv.conf" | sudo sh

# Below is temporary workaround of cross-cloud docker production image, which do
# not create RBAC as a part of deployment process. Must be removed once issue #
# resolved
git clone --depth 1 https://github.com/crosscloudci/cross-cloud.git
kubectl create -f ./cross-cloud/rbac/
# End of workaround
