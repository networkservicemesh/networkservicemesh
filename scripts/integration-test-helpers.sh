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
# Wait for all pods to become running in a $1 namespace
# Default timeout of 180 seconds can be overwritten in $2 parameter.
#
function wait_for_pods {
    set +x
    end=$(date +%s)
    if [ x$2 != "x" ]; then
     end=$((end + $2))
    else
     end=$((end + 180))
    fi
    while true; do
        kubectl get pods --namespace=$1 -o json | jq -r \
            '.items[].status.phase' | grep Pending > /dev/null && \
            PENDING=True || PENDING=False
        query='.items[]|select(.status.phase=="Running")'
        query="$query|.status.containerStatuses[].ready"
        kubectl get pods --namespace=$1 -o json | jq -r "$query" | \
            grep false > /dev/null && READY="False" || READY="True"
        kubectl get jobs -o json --namespace=$1 | jq -r \
            '.items[] | .spec.completions == .status.succeeded' | \
            grep false > /dev/null && JOBR="False" || JOBR="True"
        [ $PENDING == "False" -a $READY == "True" -a $JOBR == "True" ] && \
            break || true
        sleep 1
        now=$(date +%s)
        [ $now -gt $end ] && echo containers failed to start. && \
            kubectl get pods --namespace $1 && error_collection
    done
    set -x
}

#
# In case if a failure, collecting some evidence for further debugging
#
function error_collection {
    echo "=====> ERROR: Timed out waiting for NSM daemonset to start"
    kubectl describe node || true
    kubectl get pods --all-namespaces || true
    nsm=$(kubectl get pods --all-namespaces | grep networkservice | awk '{print $2}')
    namespace=$(kubectl get pods --all-namespaces | grep networkservice | awk '{print $1}')
    if [[ "x$nsm" != "x" ]]; then 
        kubectl describe pod $nsm -n $namespace || true
        kubectl logs $nsm -n $namespace || true
        kubectl logs $nsm -n $namespace -p || true
    fi
    nsm_client=$(kubectl get pods --all-namespaces | grep nsm-client | awk '{print $2}')
    if [[ "x$nsm_client" != "x" ]]; then 
        kubectl describe pod $nsm_client -n $namespace || true
        kubectl logs $nsm_client -n $namespace nsm-init || true
        kubectl logs $nsm_client -n $namespace nsm-client || true
        kubectl logs $nsm_client -n $namespace nsm-init -p || true
        kubectl logs $nsm_client -n $namespace nsm-client -p || true
    fi
    sudo docker images
    exit 1
}

# vim: sw=4 ts=4 et si
