#!/usr/bin/env bash

echo "Authenticate"
echo "Identify token $(gcloud auth print-identity-token)"

if gcloud auth print-identity-token 2>&1 | grep -q 'You do not currently have an active account selected'; then
    echo "Doing auth"
    echo "$GCLOUD_SERVICE_KEY" | gcloud auth activate-service-account --key-file=-
else
    echo "All is ok"
fi
