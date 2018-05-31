#!/bin/bash
#
# Sample script for creating testable Network Service Mesh CRDs
#
# NOTE: This assumes minikube for now

# First, label the nodes
kubectl label --overwrite nodes minikube app=networkservice-node

# Now create the daemonset
kubectl create -f conf/sample/networkservice-daemonset.yaml

# Create the channel
kubectl create -f conf/sample/networkservice-channel.yaml

# Create the endpoints
kubectl create -f conf/sample/networkservice-endpoint.yaml

# Finally, create the network service
kubectl create -f conf/sample/networkservice.yaml

# Dump some info
kubectl get pods,crd,NetworkService,NetworkServiceChannel,NetworkServiceEndpoint
