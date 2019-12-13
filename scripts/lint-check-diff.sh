#!/usr/bin/env bash

function check_diff() {
    GOGC=50 golangci-lint run --new-from-rev=origin/master
}

attempts=10
status=0
result=""

for (( i=0; i < "${attempts}"; i++ ))
do
    echo "Attempt $i"
    result="$(check_diff)" || status=1
    if ! [[ "${result}" == *"failed to load program with go/packages"* ]] ; then
        break
    fi
    if [[ "$((i+1))" == "${attempts}" ]] ; then
        echo "can not load program with go/packages"
        exit 1
    fi
done
echo "${result}"
exit $status