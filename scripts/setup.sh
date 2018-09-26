#!/bin/bash
#
# Sample script for creating Network Service Mesh CRDs.
#
# NOTE: This assumes minikube for now during the label step.

# First, label the nodes
kubectl label --overwrite nodes minikube app=networkservice-node

# Now create the daemonset
kubectl create -f conf/sample/networkservice-daemonset.yaml

# Create the network service
kubectl create -f conf/sample/networkservice.yaml

# Now create the NSE
kubectl create -f conf/sample/nse.yaml

# Now create the client
kubectl create -f conf/sample/nsm-client.yaml

# Dump some info
kubectl get pods,crd,NetworkService,NetworkServiceChannel,NetworkServiceEndpoint
