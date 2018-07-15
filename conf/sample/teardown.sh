#!/bin/bash
#
# Sample script for deleting Network Service Mesh CRDs.
#

here="$(dirname "$0")"

# Delete the client
kubectl delete -f "$here/conf/sample/nsm-client.yaml"

# First, delete the network service
kubectl delete -f "$here/networkservice.yaml"

# Delete the endpoints
kubectl delete -f "$here/networkservice-endpoint.yaml"

# Delete the channel
kubectl delete -f "$here/networkservice-channel.yaml"

# Now delete the daemonset
kubectl delete -f "$here/conf/sample/networkservice-daemonset.yaml"

# Now delete the CRD definitions themselves
kubectl delete crd networkservices.networkservicemesh.io
kubectl delete crd networkservicechannels.networkservicemesh.io
kubectl delete crd networkserviceendpoints.networkservicemesh.io
