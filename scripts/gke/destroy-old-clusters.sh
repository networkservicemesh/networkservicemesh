#!/bin/bash

usage() { echo "Usage: $0 [-t <hours>] [-p <string>]" 1>&2; exit 1; }
numreg='^[0-9]+$'

while getopts ":t:p:" o; do
    case "${o}" in
        p)
            p=${OPTARG}
            ;;
        t)
            t=${OPTARG}
            if ! [[ $t =~ $numreg ]] ; then
                usage
            fi
            ;;
        *)
            usage
            ;;
    esac
done
shift $((OPTIND-1))

if [ -z "${t}" ] || [ -z "${p}" ]; then
    usage
fi

CLUSTERS=$(gcloud container clusters list --format="table[no-heading](name,zone)" --filter="createTime<-PT${t}H" | grep  "${p}")
IFS=$'\n'; raws=($CLUSTERS); unset IFS;
for ((i=1;i<=${#raws[@]};i++)); do
    IFS=$' '; cols=(${raws[$i-1]}); unset IFS;
    echo "Deleting cluster ${cols[0]}"
    gcloud container clusters delete ${cols[0]} --project=${GKE_PROJECT_ID} --zone=${cols[1]} -q 1>/dev/null 2>&1 && echo "Deleted ${cols[0]}" || echo "Error deleting ${cols[0]}" &
done

wait