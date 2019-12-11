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

Usage: $0 <cluster_age_hours> <cluster_name_pattern>" 1>&2; exit 1; }
numreg='^[0-9]+$'

time_passed=$1
pattern=$2

# Check arguments are set
if [ -z "${time_passed}" ] || [ -z "${pattern}" ]; then
    usage
fi

# Check time is a number value
if ! [[ $time_passed =~ $numreg ]] ; then
    usage
fi

CLUSTERS=$(az aks list -g "nsm-ci" --query "[].{Name:name,Id:id,Group:resourceGroup}" -o tsv | grep "${pattern}")

# shellcheck disable=SC2206
IFS=$'\n' rows=($CLUSTERS); # Split result to string array by rows

for ((i=0;i<${#rows[@]};i++)); do
    cluster_info=${rows[$i]}
    
    # shellcheck disable=SC2206
    IFS=$'\t' cols=(${cluster_info}); # Split each row to values array by \t separator ([0]=name, [1]=id, [2]=resourceGroup)
    
    last_activity=$(get_date "$(get_last_cluster_activity "${cols[1]}")")
    countdown=$(get_date "-${time_passed}hours")
    
    if [[ $last_activity < $countdown ]]; then
        echo "Deleting cluster ${cols[2]} ${cols[0]} (created $last_activity)"
        "$(dirname "${BASH_SOURCE[0]}")"/destroy-aks-cluster.sh "${cols[2]}" "${cols[0]}"
    else
        echo "Skip cluster ${cols[0]} (created $last_activity)" 
    fi
done
