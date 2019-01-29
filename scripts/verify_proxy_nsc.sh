#!/bin/bash

ITERATIONS=${ITERATIONS:-3}
BATCHES=${BATCHES:-1}

function call_wget() {
    i="$1"
    nsc="$2"
    args="$3"

    if kubectl exec -it "${nsc}" -- wget "${args}" -O /dev/null --timeout 5 "localhost:8080" >/dev/null 2>&1; then
        echo "${i}. Proxy NSC accessiing 'secure-intranet-connectivity' with 'app=firewall' successful"
        exit 0
    else
        echo "Proxy NSC accessiing 'secure-intranet-connectivity' with 'app=firewall' unsuccessful"
        kubectl get pod "${nsc}" -o wide
        exit 1
    fi
}

for nsc in $(kubectl get pods -o=name | grep proxy-nsc | sed 's@.*/@@'); do
    echo "===== >>>>> PROCESSING ${nsc}  <<<<< ==========="

    # This loops and calls with "NSM-App: Firewall" header, directly into the gateway
    for ((i=1;i<=ITERATIONS;i=i+BATCHES)); do
        for ((j=i;j<i+BATCHES;++j)); do
            call_wget ${j} "${nsc}" "--header='NSM-App: Firewall'" &
            pids[${j}]=$!
        done
        # wait for all pids
        for pid in ${pids[*]}; do
            if ! wait $pid; then
                echo "A subprocess failed"
                exit 1
            fi
        done
        # sleep 1
    done
done
echo "All check OK. NSC ${nsc} behaving as expected."
exit 0