#!/bin/bash

CMD=$*
res=0

while read -r f; do
    ( d=$(dirname "$f") && pushd "$d" && $CMD; ) || res=1
done <<< "$(find . -name 'go.mod' -not -path './*vendor*/*')"

exit $res
