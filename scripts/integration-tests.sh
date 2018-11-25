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
    kubectl get nodes
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

    cp k8s/conf/vppagent-dataplane.yaml /tmp/vppagent-dataplane.yaml
    yq w -i /tmp/vppagent-dataplane.yaml spec.template.spec.containers[0].image networkservicemesh/vppagent-dataplane:"${COMMIT}"
    kubectl apply -f /tmp/vppagent-dataplane.yaml

    cp k8s/conf/nsmd.yaml /tmp/nsmd.yaml
    yq w -i /tmp/nsmd.yaml spec.template.spec.containers[0].image networkservicemesh/nsmdp:"${COMMIT}"
    yq w -i /tmp/nsmd.yaml spec.template.spec.containers[1].image networkservicemesh/nsmd:"${COMMIT}"
    yq w -i /tmp/nsmd.yaml spec.template.spec.containers[2].image networkservicemesh/nsmd-k8s:"${COMMIT}"
    kubectl apply -f /tmp/nsmd.yaml

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

    cp k8s/conf/icmp-responder-nse.yaml /tmp/icmp-responder-nse.yaml
    yq w -i /tmp/icmp-responder-nse.yaml spec.template.spec.containers[0].image networkservicemesh/icmp-responder-nse:"${COMMIT}"
    kubectl apply -f /tmp/icmp-responder-nse.yaml

    cp k8s/conf/vppagent-icmp-responder-nse.yaml /tmp/vppagent-icmp-responder-nse.yaml
    yq w -i /tmp/vppagent-icmp-responder-nse.yaml spec.template.spec.containers[0].image networkservicemesh/vppagent-icmp-responder-nse:"${COMMIT}"
    kubectl apply -f /tmp/vppagent-icmp-responder-nse.yaml

    wait_for_pods default

    typeset -i cnt=240
    until kubectl get nse | grep icmp; do
        ((cnt=cnt-1)) || return 1
        sleep 2
    done

    cp k8s/conf/nsc.yaml /tmp/nsc.yaml
    yq w -i /tmp/nsc.yaml spec.template.spec.containers[0].image networkservicemesh/nsc:"${COMMIT}"
    kubectl apply -f /tmp/nsc.yaml

    typeset -i cnt=240
    until kubectl get pods | grep nsc | grep Running ; do
        ((cnt=cnt-1)) || return 1
        sleep 2
    done

    #  Ping all the things!
    for nsc in $(kubectl get pods -o=name | grep nsc | sed 's@.*/@@'); do
        for ip in $(kubectl exec -it "${nsc}" -- ip addr| grep inet | awk '{print $2}'); do
            if [ "${ip}" = "10.20.1.1/30" ];then
                targetIp="10.20.1.2"
                endpointName="icmp-responder-nse"
            elif [ "${ip}" = "10.30.1.1/30" ];then
                targetIp="10.30.1.2"
                endpointName="vppagent-icmp-responder-nse"
            fi
            if [ ! -z ${targetIp} ]; then
                if kubectl exec -it "${nsc}" -- ping -c 1 ${targetIp} ; then
                    echo "NSC ${nsc} with IP ${ip} pinging ${endpointName} TargetIP: ${targetIp} successful"
                    PingSuccess="true"
                else
                    echo "NSC ${nsc} with IP ${ip} pinging ${endpointName} TargetIP: ${targetIp} unsuccessful"
                    return 1
                fi
                unset targetIp
                unset endpointName
            fi
        done
        if [ -z ${PingSuccess} ]; then
            echo "NSC ${nsc} failed to connect to an icmp-responder NetworkService"
            return 1
        fi
        unset PingSuccess
    done

    # We're all good now
    return 0
}

# vim: sw=4 ts=4 et si
