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

export TF_VAR_packet_project_id="${PACKET_PROJECT_ID}"
K8S_DEPLOYMENT_NAME=nsm-ci-"${CIRCLE_WORKFLOW_ID:0:8}"
./provision.sh "$1" "$K8S_DEPLOYMENT_NAME" file
