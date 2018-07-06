#!/bin/bash
#
# Sample script for deleting Network Service Mesh CRDs.
#

here="$(dirname "$0")"

# First, delete the network service
kubectl delete -f "$here/networkservice.yaml"

# Delete the endpoints
kubectl delete -f "$here/networkservice-endpoint.yaml"

# Delete the channel
kubectl delete -f "$here/networkservice-channel.yaml"

# Now delete the daemonset
kubectl delete -f "$here/networkservice-daemonset.yaml"
