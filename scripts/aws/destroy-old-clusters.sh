#!/bin/bash

usage() { echo "Cleanup aws cloud from old clusters

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

pushd "$(dirname "${BASH_SOURCE[0]}")" || exit 1
AWS_REGION=us-east-2 go run ./... DeleteAll "${time_passed}" "${pattern}"
popd || exit 0