#!/bin/bash
#
# Sample script for creating Network Service Mesh CRDs.
#
# NOTE: This assumes minikube for now during the label step.

here="$(dirname "$0")"

# First, label the nodes.  These are the nodes on which the service will run.
kubectl label --overwrite nodes minikube app=networkservice-node

# Now create the daemonset.  This will run the containers.
# It will also create the service types.
kubectl create -f "$here/networkservice-daemonset.yaml"

# Creating the daemonset creates the CRDs for the NSM-specific resources.
# This can take a little time.
for f in $(seq 1 20); do
    sleep 3 # 60s all told, plus command time
    if kubectl get crd networkservices.networkservicemesh.io >& /dev/null; then
        ready=1
        break
    fi
done

if [ -z "$ready" ] ; then
    echo "DaemonSet did not come up."
    echo "One common reason is that the NSM container image is not in"
    echo "the registry that your k8s is using.  Please ensure it has"
    echo "been uploaded there."
    exit 1
fi

# Create the channel
kubectl create -f "$here/networkservice-channel.yaml"

# Create the endpoints
kubectl create -f "$here/networkservice-endpoint.yaml"

# Finally, create the network service
kubectl create -f "$here/networkservice.yaml"

# Dump some info
kubectl get pods,crd,NetworkService,NetworkServiceChannel,NetworkServiceEndpoint
