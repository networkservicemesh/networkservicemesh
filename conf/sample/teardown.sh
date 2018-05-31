#!/bin/bash
#
# Sample script for creating testable Network Service Mesh CRDs
#
# NOTE: This assumes minikube for now

# First, delete the network service
kubectl delete -f conf/sample/networkservice.yaml

# Delete the endpoints
kubectl delete -f conf/sample/networkservice-endpoint.yaml

# Delete the channel
kubectl delete -f conf/sample/networkservice-channel.yaml

# Now delete the daemonset
kubectl delete -f conf/sample/networkservice-daemonset.yaml
