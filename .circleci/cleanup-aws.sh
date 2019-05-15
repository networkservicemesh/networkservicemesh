#!/usr/bin/env bash

./scripts/aws/aws-init.sh
kubectl delete svc --all --all-namespaces
export NSM_AWS_SERVICE_SUFFIX="-${CLUSTER_ID}-${CIRCLE_WORKFLOW_ID}"
make aws-destroy
