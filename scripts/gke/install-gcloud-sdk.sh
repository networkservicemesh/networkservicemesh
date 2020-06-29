#!/bin/bash
CLOUD_SDK_REPO="cloud-sdk-stretch"
export CLOUD_SDK_REPO
echo "deb http://packages.cloud.google.com/apt $CLOUD_SDK_REPO main" | tee -a /etc/apt/sources.list.d/google-cloud-sdk.list
curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -
apt-get update -y --allow-unauthenticated && apt-get install google-cloud-sdk -y --allow-unauthenticated