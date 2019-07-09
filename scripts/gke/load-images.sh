#!/bin/bash

if [ -z "$1" ]
then
    echo "No image supplied"
    exit 1
fi

if [ ! -f "scripts/vagrant/images/$1" ]
then
    echo "Image $1 is not found"
    exit 1
fi

GKE_GROUP=$(gcloud container clusters describe dev --project="$GKE_PROJECT_ID" --zone "$GKE_CLUSTER_ZONE" --format=text | grep "^instanceGroupUrls\[0\]" | sed 's/^instanceGroupUrls\[0\]:.*\///g')
GKE_INSTANCES=$(gcloud compute instance-groups list-instances --project="$GKE_PROJECT_ID" --zone "$GKE_CLUSTER_ZONE" "$GKE_GROUP" | tail -n +2 | sed 's/ .*//g')
echo "gke instances: $GKE_INSTANCES"
for NODE in $GKE_INSTANCES 
do
  echo "Loading image $1 to $NODE:"
  gcloud compute scp --zone="$GKE_CLUSTER_ZONE" "scripts/vagrant/images/$1" "$NODE:$HOME/"
  gcloud compute ssh --zone="$GKE_CLUSTER_ZONE" "$NODE" --command="sudo docker load -i ~/$1"
done