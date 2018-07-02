#!/bin/bash
#
# Sample script for deleting Network Service Mesh CRDs.
#

# Delete the client
kubectl delete -f conf/sample/nsm-client.yaml

# First, delete the network service
kubectl delete -f conf/sample/networkservice.yaml

# Delete the endpoints
kubectl delete -f conf/sample/networkservice-endpoint.yaml

# Delete the channel
kubectl delete -f conf/sample/networkservice-channel.yaml

# Now delete the daemonset
kubectl delete -f conf/sample/networkservice-daemonset.yaml

# Now delete the CRD definitions themselves
kubectl delete crd networkservices.networkservicemesh.io
kubectl delete crd networkservicechannels.networkservicemesh.io
kubectl delete crd networkserviceendpoints.networkservicemesh.io
