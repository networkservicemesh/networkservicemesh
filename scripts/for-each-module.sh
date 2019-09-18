#!/bin/bash

res=0
cmd=$1

exec_command () {
    d=$(dirname "$0")
    pushd "$d" || exit 1
    
    if ! $1; then
        res=1
    fi

    popd || exit 1
}

export -f exec_command
find . -name "go.mod" -exec bash -c "exec_command \"${cmd}\" " {} \;

exit $res
