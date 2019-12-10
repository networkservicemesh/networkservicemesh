#!/bin/bash

usage() { echo "Cleanup gke cloud from old clusters

Usage: $0 [-t <hours>] [-p <string>]

Flags:
  -t    Time has passed since the creation of the cluster
  -p    Cluster name pattern
" 1>&2; exit 1; }
numreg='^[0-9]+$'

while getopts ":t:p:" o; do
    case "${o}" in
        p)
            pattern=${OPTARG}
            ;;
        t)
            time_passed=${OPTARG}
            if ! [[ $time_passed =~ $numreg ]] ; then
                usage
            fi
            ;;
        *)
            usage
            ;;
    esac
done
shift $((OPTIND-1))

if [ -z "${time_passed}" ] || [ -z "${pattern}" ]; then
    usage
fi

CLUSTERS=$(gcloud container clusters list --project="$GKE_PROJECT_ID" --format="table[no-heading](name,zone,createTime)" --filter="createTime<-PT${time_passed}H" | grep  "${pattern}")
IFS=$'\n'; 
# shellcheck disable=SC2206
raws=($CLUSTERS); 
unset IFS;
for ((i=1;i<=${#raws[@]};i++)); do
    IFS=$' '; read -r -a cols <<< "${raws[$i-1]}"; unset IFS;
    echo "Deleting cluster ${cols[0]} (created ${cols[2]})"
    gcloud container clusters delete "${cols[0]}" --project="${GKE_PROJECT_ID}" --zone="${cols[1]}" -q 1>/dev/null 2>&1 && echo "Deleted ${cols[0]}" || echo "Error deleting ${cols[0]}" &
done

wait