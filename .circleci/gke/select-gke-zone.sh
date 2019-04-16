#!/usr/bin/env bash
ZONES=$(gcloud compute zones list --uri | grep -v asia | grep -v australia | cut -f 9 -d '/')
ZONE_COUNT=$(echo "$ZONES" | wc -l | xargs)
VALUE=$1
INDEX=$((VALUE % ZONE_COUNT + 1))
echo "${ZONES}" | tail -n "${INDEX}" | head -n 1
