#!/usr/bin/env bash

echo "Authenticate if not yet"
if gcloud auth print-identity-token 2>&1 | grep -q 'gcloud config set account ACCOUNT'; then
    echo "Doing auth"
    echo "$GCLOUD_SERVICE_KEY" | gcloud auth activate-service-account --key-file=-
else
    echo "All is ok"
fi
