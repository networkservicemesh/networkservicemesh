#!/bin/bash
#
# Sample script for deleting Network Service Mesh CRDs.
#

# First, delete the network service
kubectl delete -f conf/sample/networkservice.yaml

# Delete the endpoints
kubectl delete -f conf/sample/networkservice-endpoint.yaml

# Delete the channel
kubectl delete -f conf/sample/networkservice-channel.yaml

# Now delete the daemonset
kubectl delete -f conf/sample/networkservice-daemonset.yaml
