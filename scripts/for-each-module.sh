#!/bin/bash

CMD=$*
res=0

while read -r f; do
    ( d=$(dirname "$f") && pushd "$d" && $CMD; ) || res=1
done <<< "$(find . -path "*vendor" -prune -o -name 'go.mod')"

exit $res
