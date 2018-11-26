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

export TF_VAR_auth_token="${PACKET_AUTH_TOKEN}"
export TF_VAR_master_hostname="ci-${CIRCLE_BUILD_NUM}-master"
export TF_VAR_worker1_hostname="ci-${CIRCLE_BUILD_NUM}-worker1"
export TF_VAR_project_id="${PACKET_PROJECT_ID}"
export TF_VAR_public_key="${PWD}/data/sshkey.pub"
export TF_VAR_public_key_name="key-${CIRCLE_BUILD_NUM}"

echo "workdir: ${PWD}"
make packet-init
make packet-start
#./provision.sh "$1" "$K8S_DEPLOYMENT_NAME" file
