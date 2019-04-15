#!/usr/bin/env bash
export GCLOUD_SERVICE_KEY=$1
export COMMIT=$2
export GOOGLE_PROJECT_ID=$3
export GOOGLE_COMPUTE_ZONE=$4

sudo ./scripts/install-kubectl.sh
sudo ./scripts/gke/install-gcloud-sdk.sh

echo "$GCLOUD_SERVICE_KEY" | gcloud auth activate-service-account --key-file=-
gcloud config set project "${GOOGLE_PROJECT_ID}"
gcloud config set compute/zone "${GOOGLE_COMPUTE_ZONE}"