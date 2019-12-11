#!/bin/bash

# A script to run through all the modules and execute the supplied command(s)
# The name of the module can be accessed in the command sith _MODULE_ variable.
# Example: for-each-module.sh echo \$_MODULE_

CMD="$*"
res=0

while read -r f; do
    ( d=$(dirname "$f") && pushd "$d" && export _MODULE_=${d#\.\/} && eval "$CMD"; ) || res=1
done <<< "$(find . -name 'go.mod' -not -path './*vendor*/*')"

exit $res
