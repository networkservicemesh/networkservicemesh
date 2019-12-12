#!/bin/bash

usage() {
    echo -e "Cleanup gke cloud from old clusters.\n\nUsage: $0 <cluster_age_hours> <cluster_name_pattern>" 1>&2
    exit 1
}

numreg='^[0-9]+$'

max_cluster_age=$1
pattern=$2

# Check arguments are set
if [ -z "${max_cluster_age}" ] || [ -z "${pattern}" ]; then
    usage
fi

# Check time is a number value
if ! [[ $max_cluster_age =~ $numreg ]] ; then
    usage
fi

CLUSTERS=$(gcloud container clusters list --project="$GKE_PROJECT_ID" --format="table[no-heading](name,zone,createTime)" --filter="createTime<-PT${max_cluster_age}H" | grep  "${pattern}")

# shellcheck disable=SC2206
IFS=$'\n' rows=($CLUSTERS); # Split result to string array by rows

for cluster_info in "${rows[@]}"; do
    # shellcheck disable=SC2206
    IFS=$' ' cols=(${cluster_info}); # Split each row to values array by space ([0]=name, [1]=zone, [2]=createTime)
    echo "Deleting cluster ${cols[0]} (created ${cols[2]})"
    gcloud container clusters delete "${cols[0]}" --project="${GKE_PROJECT_ID}" --zone="${cols[1]}" -q 1>/dev/null 2>&1 && echo "Deleted ${cols[0]}" || echo "Error deleting ${cols[0]}" &
done

wait