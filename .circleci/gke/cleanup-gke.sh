#!/usr/bin/env bash

export GCLOUD_SERVICE_KEY=$1
export COMMIT=$2
export GOOGLE_PROJECT_ID=$3
export CIRCLE_BUILD_NUM=$4
export GKE_CLUSTER_NAME=$5

export CONTAINER_TAG="${COMMIT}"

.circleci/gke/init-gke.sh "$GCLOUD_SERVICE_KEY" "$COMMIT" "$GOOGLE_PROJECT_ID" "$CIRCLE_BUILD_NUM" "$GKE_CLUSTER_NAME"
make gke-destroy