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

pushd "$(dirname "${BASH_SOURCE[0]}")" || exit 1
AWS_REGION=us-east-2 go run ./... DeleteAll "${t}" "${p}"
popd || exit 0