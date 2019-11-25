#!/bin/bash

kubectl="kubectl -n ${NSM_NAMESPACE:-default}"

#  Ping all the things!
EXIT_VAL=0
for nsc in $(${kubectl} get pods -o=name | grep -E "icmp-responder-nsc|vpp-icmp-responder-nsc" | sed 's@.*/@@'); do
    echo "===== >>>>> PROCESSING ${nsc}  <<<<< ==========="
    if [[ ${nsc} == vpp-* ]]; then
        for ip in $(${kubectl} exec "${nsc}" -- vppctl show int addr | grep L3 | awk '{print $2}'); do
            if [[ "${ip}" == 172.16.1.* ]];then
                lastSegment=$(echo "${ip}" | cut -d . -f 4 | cut -d / -f 1)
                nextOp=$((lastSegment + 1))
                targetIp="172.16.1.${nextOp}"
                endpointName="icmp-responder-nse"
            elif [[ "${ip}" == 10.30.1.* ]];then
                lastSegment=$(echo "${ip}" | cut -d . -f 4 | cut -d / -f 1)
                nextOp=$((lastSegment + 1))
                targetIp="10.30.1.${nextOp}"
                endpointName="vpp-icmp-responder-nse"
            fi

            if [ -n "${targetIp}" ]; then
                # Prime the pump, its normal to get a packet loss due to arp
                ${kubectl} exec "${nsc}" -- vppctl ping "${targetIp}" repeat 1 > /dev/null 2>&1
                OUTPUT=$(${kubectl} exec "${nsc}" -- vppctl ping "${targetIp}" repeat 3)
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
        for ip in $(${kubectl} exec "${nsc}" -- ip addr| grep inet | awk '{print $2}'); do
            if [[ "${ip}" == 172.16.1.* ]];then
                lastSegment=$(echo "${ip}" | cut -d . -f 4 | cut -d / -f 1)
                nextOp=$((lastSegment + 1))
                targetIp="172.16.1.${nextOp}"
                endpointName="icmp-responder-nse"
            elif [[ "${ip}" == 10.30.1.* ]];then
                lastSegment=$(echo "${ip}" | cut -d . -f 4 | cut -d / -f 1)
                nextOp=$((lastSegment + 1))
                targetIp="10.30.1.${nextOp}"
                endpointName="vpp-icmp-responder-nse"
            fi

            if [ -n "${targetIp}" ]; then
                if ${kubectl} exec "${nsc}" -- ping -c 1 "${targetIp}" ; then
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
        if [[ ${nsc} == vpp-* ]]; then
            ${kubectl} exec "${nsc}" -- vppctl show int
            ${kubectl} exec "${nsc}" -- vppctl show int addr
            ${kubectl} exec "${nsc}" -- vppctl show memif
        else
            ${kubectl} exec "${nsc}" -- ip addr
            ${kubectl} exec "${nsc}" ip route
        fi
        echo "+++++++==ERROR==ERROR=============================================================================+++++"
    fi
    unset PingSuccess
done
exit ${EXIT_VAL}
