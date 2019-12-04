#!/bin/bash
get_date() {
    gdate --utc --date="$1" +"%Y-%m-%dT%H:%M:%S"
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

CLUSTERS=$(az aks list -g "nsm-ci" --query "[].{Name:name,Id:id,Group:resourceGroup}" -o tsv | grep "${p}")
IFS=$'\n'; 
# shellcheck disable=SC2206
raws=($CLUSTERS); 
unset IFS;
for ((i=1;i<=${#raws[@]};i++)); do
    IFS=$'\t'; read -r -a cols <<< "${raws[$i-1]}"; unset IFS;
    last_activity=$(get_date "$(get_last_cluster_activity "${cols[1]}")")
    countdown=$(get_date "-${t}hours")
    if [[ $last_activity < $countdown ]]; then
        echo "Deleting cluster ${cols[2]} ${cols[0]} (created $last_activity)"
        "$(dirname "${BASH_SOURCE[0]}")"/destroy-aks-cluster.sh "${cols[2]}" "${cols[0]}"
    else
        echo "Skip cluster ${cols[0]} (created $last_activity)" 
    fi
done

