#!/bin/bash

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

test -n "${COMMIT}" && set -- "${COMMIT}"

export TAG="$1"

TMPL_DIR=./conf/sample
CONF_DIR=./conf/sample

envsubst < "${TMPL_DIR}"/networkservice-daemonset.tmpl.yaml > "${CONF_DIR}"/networkservice-daemonset.yaml
envsubst < "${TMPL_DIR}"/nse.tmpl.yaml > "${CONF_DIR}"/nse.yaml
envsubst < "${TMPL_DIR}"/nsm-client.tmpl.yaml > "${CONF_DIR}"/nsm-client.yaml
envsubst < "${TMPL_DIR}"/test-dataplane.tmpl.yaml > "${CONF_DIR}"/test-dataplane.yaml
