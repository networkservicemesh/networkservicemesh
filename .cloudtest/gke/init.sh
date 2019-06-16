#!/usr/bin/env bash

echo "Authenticate"
echo "$GCLOUD_SERVICE_KEY" | gcloud auth activate-service-account --key-file=-
