#!/usr/bin/env bash
>&2 echo "Selecting zones $1"
gcloud compute zones list --uri --project="$1" | grep -v asia | grep -v australia | cut -f 9 -d '/'