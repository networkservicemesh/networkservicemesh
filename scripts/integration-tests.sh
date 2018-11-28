#!/bin/bash

# Copyright (c) 2016-2017 Bitnami
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

. scripts/integration-test-helpers.sh

function run_tests() {
    COMMIT=${COMMIT:-latest}
    kubectl get nodes -o wide
    kubectl version
    kubectl api-versions
    kubectl label --overwrite --all=true nodes app=nsmd-ds
    #kubectl label --overwrite nodes kube-node-1 app=networkservice-node
    #
    # Now let's wait for all pods to get into running state
    #
    wait_for_pods default
    exit_code=$?
    [[ ${exit_code} != 0 ]] && return ${exit_code}

    kubectl apply -f k8s/conf/cluster-role-admin.yaml
    kubectl apply -f k8s/conf/cluster-role-binding.yaml

    make k8s-vppagent-dataplane-deploy
    make k8s-nsmd-deploy
    make k8s-crossconnect-monitor-deploy

    # Wait til settles
    echo "INFO: Waiting for Network Service Mesh daemonset to be up and CRDs to be available ..."
    typeset -i cnt=240
    until kubectl get crd | grep networkserviceendpoints.networkservicemesh.io ; do
        ((cnt=cnt-1)) || return 1
        sleep 2
    done
    typeset -i cnt=240
    until kubectl get crd | grep networkservices.networkservicemesh.io ; do
        ((cnt=cnt-1)) || return 1
        sleep 2
    done

    wait_for_pods default

    make k8s-icmp-responder-nse-deploy
    make k8s-vppagent-icmp-responder-nse-deploy

    wait_for_pods default

    typeset -i cnt=240
    until kubectl get nse | grep icmp; do
        ((cnt=cnt-1)) || return 1
        sleep 2
    done

    make k8s-nsc-deploy

    wait_for_pods default

    # We're all good now
    return 0
}

# vim: sw=4 ts=4 et si
