#!/bin/bash
get_date() {
    date --utc --date="$1" +"%Y-%m-%dT%H:%M:%S"
}
get_last_cluster_activity() {
    az monitor activity-log list \
        --max-events 1 \
        --select eventTimestamp resourceId \
        --status "Started" \
        --resource-id "$1" \
        --query "[].{Tiem:eventTimestamp}" \
        -o tsv
}
usage() { echo "Cleanup azure cloud from old clusters

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

CLUSTERS=$(az aks list -g "nsm-ci" --query "[].{Name:name,Id:id,Group:resourceGroup}" -o tsv | grep "${pattern}")
IFS=$'\n'; 
# shellcheck disable=SC2206
raws=($CLUSTERS); 
unset IFS;
for ((i=1;i<=${#raws[@]};i++)); do
    IFS=$'\t'; read -r -a cols <<< "${raws[$i-1]}"; unset IFS;
    last_activity=$(get_date "$(get_last_cluster_activity "${cols[1]}")")
    countdown=$(get_date "-${time_passed}hours")
    if [[ $last_activity < $countdown ]]; then
        echo "Deleting cluster ${cols[2]} ${cols[0]} (created $last_activity)"
        "$(dirname "${BASH_SOURCE[0]}")"/destroy-aks-cluster.sh "${cols[2]}" "${cols[0]}"
    else
        echo "Skip cluster ${cols[0]} (created $last_activity)" 
    fi
done

