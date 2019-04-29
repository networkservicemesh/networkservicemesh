#!/bin/bash

kubectl="kubectl -n ${NSM_NAMESPACE}"

#  Ping all the things!
EXIT_VAL=0
for nsc in $(${kubectl} get pods -o=name | grep nsc | sed 's@.*/@@'); do
    echo "POD ${nsc} Network dump ${nsc} -------------------------------"
    #${kubectl} exec -ti "${nsc}" ip addr
    ${kubectl} exec -ti "${nsc}" ip route
done
exit ${EXIT_VAL}