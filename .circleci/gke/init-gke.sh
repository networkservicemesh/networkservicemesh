#!/usr/bin/env bash
export GCLOUD_SERVICE_KEY=$1
export COMMIT=$2
export GOOGLE_PROJECT_ID=$3
export CIRCLE_BUILD_NUM=$4

sudo ./scripts/install-kubectl.sh
sudo ./scripts/gke/install-gcloud-sdk.sh

echo "$GCLOUD_SERVICE_KEY" | gcloud auth activate-service-account --key-file=-
gcloud config set project "${GOOGLE_PROJECT_ID}"
GOOGLE_COMPUTE_ZONE=$(./.circleci/gke/select-gke-zone.sh "$CIRCLE_BUILD_NUM")
gcloud config set compute/zone "${GOOGLE_COMPUTE_ZONE}"