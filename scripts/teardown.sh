#!/bin/bash
#
# Sample script for deleting Network Service Mesh CRDs.
#

# Delete the client
kubectl delete -f conf/sample/nsm-client.yaml

# Delete the dataplane
kubectl delete -f conf/sample/test-dataplane.yaml

# Delete the NSE
kubectl delete -f conf/sample/nse.yaml

# First, delete the network service
kubectl delete -f conf/sample/networkservice.yaml

# Now delete the daemonset
kubectl delete -f conf/sample/networkservice-daemonset.yaml

# Now delete the CRD definitions themselves
kubectl delete crd networkservices.networkservicemesh.io
kubectl delete crd networkserviceendpoints.networkservicemesh.io
