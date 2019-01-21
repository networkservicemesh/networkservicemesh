#!/bin/bash

ITERATIONS=${ITERATIONS:-3}
BATCHES=3
# The direct link of Proxy-NSC to "secure-intranet-connectivity" is disabled for issues
#DIRECT:=1

function call_wget() {
    i="$1"
    nsc="$2"
    args="$3"

    if kubectl exec -it "${nsc}" -- wget "${args}" -O /dev/null --timeout 3 "localhost:8080"; then # 2>&1 >/dev/null; then
        echo "${i}. Proxy NSC accessiing 'secure-intranet-connectivity' with 'app=firewall' successful"
    else
        echo "Proxy NSC accessiing 'secure-intranet-connectivity' with 'app=firewall' unsuccessful"
        kubectl get pod "${nsc}" -o wide
        exit 1
    fi
}

for nsc in $(kubectl get pods -o=name | grep proxy-nsc | sed 's@.*/@@'); do
    echo "===== >>>>> PROCESSING ${nsc}  <<<<< ==========="

    if [ -n "${DIRECT}" ]; then
        if kubectl exec -it "${nsc}" -- wget -O /dev/null --timeout 10 "localhost:8080" ; then
            echo "Proxy NSC accessiing 'secure-intranet-connectivity' successful"
        else
            echo "Proxy NSC accessiing 'secure-intranet-connectivity' unsuccessful"
            kubectl get pod "${nsc}" -o wide
        fi
    fi

    # This loops and calls with "NSM-App: Firewall" header, directly into the gateway
    for ((i=1;i<=${ITERATIONS};++i)); do
        call_wget ${i} ${nsc} "--header='NSM-App: Firewall'"
    done
done

echo "All check OK. NSC ${nsc} behaving as expected."
exit 0