#!/usr/bin/env bash

export GCLOUD_SERVICE_KEY=$1
export COMMIT=$2
export GOOGLE_PROJECT_ID=$3
export GOOGLE_COMPUTE_ZONE=$4
export GKE_CLUSTER_NAME=$5

export CONTAINER_TAG="${COMMIT}"

.circleci/gke/init-gke.sh "$GCLOUD_SERVICE_KEY" "$COMMIT" "$GOOGLE_PROJECT_ID" "$GOOGLE_COMPUTE_ZONE" "$GKE_CLUSTER_NAME"
make gke-destroy