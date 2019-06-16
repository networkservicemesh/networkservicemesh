#!/usr/bin/env bash
gcloud compute zones list --uri | grep -v asia | grep -v australia | cut -f 9 -d '/'