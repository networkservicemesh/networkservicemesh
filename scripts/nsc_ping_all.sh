#!/bin/bash

kubectl="kubectl -n ${NSM_NAMESPACE}"

#  Ping all the things!
EXIT_VAL=0
for nsc in $(${kubectl} get pods -o=name | grep -E "alpine-nsc|vppagent-nsc" | sed 's@.*/@@'); do
    echo "===== >>>>> PROCESSING ${nsc}  <<<<< ==========="
    if [[ ${nsc} == vppagent-* ]]; then
        for ip in $(${kubectl} exec -it "${nsc}" -- vppctl show int addr | grep L3 | awk '{print $2}'); do
            if [[ "${ip}" == 10.20.1.* ]];then
                lastSegment=$(echo "${ip}" | cut -d . -f 4 | cut -d / -f 1)
                nextOp=$((lastSegment + 1))
                targetIp="10.20.1.${nextOp}"
                endpointName="icmp-responder-nse"
            elif [[ "${ip}" == 10.30.1.* ]];then
                lastSegment=$(echo "${ip}" | cut -d . -f 4 | cut -d / -f 1)
                nextOp=$((lastSegment + 1))
                targetIp="10.30.1.${nextOp}"
                endpointName="vppagent-icmp-responder-nse"
            fi

            if [ -n "${targetIp}" ]; then
                # Prime the pump, its normal to get a packet loss due to arp
                ${kubectl} exec -it "${nsc}" -- vppctl ping "${targetIp}" repeat 1 > /dev/null 2>&1
                OUTPUT=$(${kubectl} exec -it "${nsc}" -- vppctl ping "${targetIp}" repeat 3)
                echo "${OUTPUT}"
                RESULT=$(echo "${OUTPUT}"| grep "packet loss" | awk '{print $6}')
                if [ "${RESULT}" = "0%" ]; then
                    echo "NSC ${nsc} with IP ${ip} pinging ${endpointName} TargetIP: ${targetIp} successful"
                    PingSuccess="true"
                else
                    echo "NSC ${nsc} with IP ${ip} pinging ${endpointName} TargetIP: ${targetIp} unsuccessful"
                    EXIT_VAL=1
                fi
                unset targetIp
                unset endpointName
            fi
        done
    else
        for ip in $(${kubectl} exec -it "${nsc}" -- ip addr| grep inet | awk '{print $2}'); do
            if [[ "${ip}" == 10.20.1.* ]];then
                lastSegment=$(echo "${ip}" | cut -d . -f 4 | cut -d / -f 1)
                nextOp=$((lastSegment + 1))
                targetIp="10.20.1.${nextOp}"
                endpointName="icmp-responder-nse"
            elif [[ "${ip}" == 10.30.1.* ]];then
                lastSegment=$(echo "${ip}" | cut -d . -f 4 | cut -d / -f 1)
                nextOp=$((lastSegment + 1))
                targetIp="10.30.1.${nextOp}"
                endpointName="vppagent-icmp-responder-nse"
            fi

            if [ -n "${targetIp}" ]; then
                if ${kubectl} exec -it "${nsc}" -- ping -c 1 "${targetIp}" ; then
                    echo "NSC ${nsc} with IP ${ip} pinging ${endpointName} TargetIP: ${targetIp} successful"
                    PingSuccess="true"
                else
                    echo "NSC ${nsc} with IP ${ip} pinging ${endpointName} TargetIP: ${targetIp} unsuccessful"
                    EXIT_VAL=1
                fi
                unset targetIp
                unset endpointName
            fi
        done
    fi
    if [ -z ${PingSuccess} ]; then
        EXIT_VAL=1
        echo "+++++++==ERROR==ERROR=============================================================================+++++"
        echo "NSC ${nsc} failed to connect to an icmp-responder NetworkService"
        ${kubectl} get pod "${nsc}" -o wide
        echo "POD ${nsc} Network dump -------------------------------"
        if [[ ${nsc} == vppagent-* ]]; then
            if [[ ${STORE_POD_LOGS_DIR} ]]; then
                if [[ ${STORE_POD_LOGS_IN_FILES} ]]; then
                    ${kubectl} logs "${nsc}" >> STORE_LOGS_DIR/example/"${nsc}".log
                else
                    ${kubectl} logs "${nsc}"
                fi
            fi
            ${kubectl} exec -ti "${nsc}" -- vppctl show int
            ${kubectl} exec -ti "${nsc}" -- vppctl show int addr
            ${kubectl} exec -ti "${nsc}" -- vppctl show memif
        else
            ${kubectl} exec -ti "${nsc}" -- ip addr
            ${kubectl} exec -ti "${nsc}" ip route
        fi
        echo "+++++++==ERROR==ERROR=============================================================================+++++"
    fi
    unset PingSuccess
done
exit ${EXIT_VAL}