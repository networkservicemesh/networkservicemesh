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


#
# Wait for network service object to become available in a $1 namespace
# Default timeout of 60 seconds can be overwritten in $2 parameter.
#
function wait_for_networkservice() {
    set +xe
    end=$(date +%s)
    if [ "x$2" != "x" ]; then
     end=$((end + "$2"))
    else
     end=$((end + 60))
    fi
    while true; do
        network_service=$(kubectl get networkservices --namespace="$1" -o json | jq -r '.items[].metadata.name')
        if [ "x$network_service" != "x" ]; then
            break
        fi
        sleep 1
        now=$(date +%s)
        if [ "$now" -gt "$end" ] ; then
            echo "NetworkService has not been created within 60 seconds, failing..."
            error_collection
            set -xe
            return 1
        fi
    done

    set -xe
    return 0
}

#
# Wait for all pods to become running in a $1 namespace
# Default timeout of 180 seconds can be overwritten in $2 parameter.
#
function wait_for_pods() {
    set +xe
    end=$(date +%s)
    if [ "x$2" != "x" ]; then
     end=$((end + "$2"))
    else
     end=$((end + 180))
    fi
    while true; do
        kubectl get pods --namespace="$1" -o json | jq -r \
            '.items[].status.phase' | grep Pending > /dev/null && \
            PENDING=True || PENDING=False
        query='.items[]|select(.status.phase=="Running")'
        query="$query|.status.containerStatuses[].ready"
        kubectl get pods --namespace="$1" -o json | jq -r "$query" | \
            grep false > /dev/null && READY="False" || READY="True"
        kubectl get jobs -o json --namespace="$1" | jq -r \
            '.items[] | .spec.completions == .status.succeeded' | \
            grep false > /dev/null && JOBR="False" || JOBR="True"
        if [ "$PENDING" == "False" ] && [ "$READY" == "True" ] && [ "$JOBR" == "True" ]
        then
            break
        fi
        sleep 1
        now=$(date +%s)
        if [ "$now" -gt "$end" ] ; then
            echo "Containers failed to start."
            error_collection
            set -xe
            return 1
        fi
    done

    set -xe
    return 0
}

#
# In case if a failure, collecting some evidence for further debugging
#
function error_collection() {
    kubectl describe node || true
    kubectl get pods --all-namespaces || true
    nsm=$(kubectl get pods --all-namespaces | grep networkservice | awk '{print $2}')
    namespace=$(kubectl get pods --all-namespaces | grep networkservice | awk '{print $1}')
    if [[ "x$nsm" != "x" ]]; then 
        kubectl describe pod "$nsm" -n "$namespace" || true
        kubectl logs "$nsm" -n "$namespace" || true
        kubectl logs "$nsm" -n "$namespace" -p || true
    fi
    nsm_client=$(kubectl get pods --all-namespaces | grep nsm-client | awk '{print $2}')
    if [[ "x$nsm_client" != "x" ]]; then 
        kubectl describe pod "$nsm_client" -n "$namespace" || true
        kubectl logs "$nsm_client" -n "$namespace" nsm-init || true
        kubectl logs "$nsm_client" -n "$namespace" nsm-client || true
        kubectl logs "$nsm_client" -n "$namespace" nsm-init -p || true
        kubectl logs "$nsm_client" -n "$namespace" nsm-client -p || true
    fi
    nse=$(kubectl get pods --all-namespaces | grep nse | awk '{print $2}')
    if [[ "x$nse" != "x" ]]; then 
        kubectl describe pod "$nse" -n "$namespace" || true
        kubectl logs "$nse" -n "$namespace"  || true
        kubectl logs "$nse" -n "$namespace"  -p || true
    fi    
    dataplane=$(kubectl get pods --all-namespaces | grep simple-dataplane | awk '{print $2}')
    if [[ "x$dataplane" != "x" ]]; then 
        kubectl describe pod "$dataplane" -n "$namespace" || true
        kubectl logs "$dataplane" -n "$namespace"  || true
        kubectl logs "$dataplane" -n "$namespace"  -p || true
    fi 
    sidecar=$(kubectl get pods --all-namespaces | grep sidecar-injector-webhook | awk '{print $2}')
    if [[ "x$sidecar" != "x" ]]; then
        kubectl describe pod "$sidecar" -n "$namespace" || true
        kubectl logs "$sidecar" -n "$namespace"  || true
        kubectl logs "$sidecar" -n "$namespace"  -p || true
    fi
    kubectl get nodes --show-label
    sudo docker images
}

# vim: sw=4 ts=4 et si
