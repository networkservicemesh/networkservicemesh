#!/bin/bash

kubectl="kubectl -n ${NSM_NAMESPACE}"

#  Ping all the things!
EXIT_VAL=0
for nsc in $(${kubectl} get pods -o=name | grep vpn-gateway-nsc | sed 's@.*/@@'); do
    echo "===== >>>>> PROCESSING ${nsc}  <<<<< ==========="
    for ip in $(${kubectl} exec -it "${nsc}" -- ip addr| grep inet | awk '{print $2}'); do
        if [[ "${ip}" == 172.16.1.* ]];then
            lastSegment=$(echo "${ip}" | cut -d . -f 4 | cut -d / -f 1)
            nextOp=$((lastSegment + 1))
            targetIp="172.16.1.${nextOp}"
            endpointName="vpn-gateway-nse"
        fi

        if [ -n "${targetIp}" ]; then


            if ${kubectl} exec -it "${nsc}" -- ping -c 1 "${targetIp}" ; then
                echo "NSC ${nsc} with IP ${ip} pinging ${endpointName} TargetIP: ${targetIp} successful"
                PingSuccess="true"
            else
                echo "NSC ${nsc} with IP ${ip} pinging ${endpointName} TargetIP: ${targetIp} unsuccessful"
                EXIT_VAL=1
            fi

            if ${kubectl} exec -it "${nsc}" -- wget -O /dev/null --timeout 3 "${targetIp}:80" ; then
                echo "NSC ${nsc} with IP ${ip} accessing ${endpointName} TargetIP: ${targetIp} TargetPort:80 successful"
                Wget80Success="true"
            else
                echo "NSC ${nsc} with IP ${ip} accessing ${endpointName} TargetIP: ${targetIp} TargetPort:80 unsuccessful"
                EXIT_VAL=1
            fi

            if ${kubectl} exec -it "${nsc}" -- wget -O /dev/null --timeout 3 "${targetIp}:8080" ; then
                echo "NSC ${nsc} with IP ${ip} accessing ${endpointName} TargetIP: ${targetIp} TargetPort:8080 successful"
                EXIT_VAL=1
            else
                echo "NSC ${nsc} with IP ${ip} blocked ${endpointName} TargetIP: ${targetIp} TargetPort:8080"
                Wget8080Blocked="true"                
            fi

            unset targetIp
            unset endpointName
        fi
    done
    if [ -z ${PingSuccess} ]; then
        EXIT_VAL=1
        echo "+++++++==ERROR==ERROR=============================================================================+++++"
        echo "NSC ${nsc} failed ping to a vpn-gateway NetworkService"
        ${kubectl} get pod "${nsc}" -o wide
        echo "POD ${nsc} Network dump -------------------------------"
        ${kubectl} exec -ti "${nsc}" -- ip addr
        echo "+++++++==ERROR==ERROR=============================================================================+++++"
    fi
    if [ -z ${Wget80Success} ]; then
        EXIT_VAL=1
        echo "+++++++==ERROR==ERROR=============================================================================+++++"
        echo "NSC ${nsc} failed to wget on port 80 to a vpn-gateway NetworkService"
        ${kubectl} get pod "${nsc}" -o wide
        echo "+++++++==ERROR==ERROR=============================================================================+++++"
    fi
    if [ -z ${Wget8080Blocked} ]; then
        EXIT_VAL=1
        echo "+++++++==ERROR==ERROR=============================================================================+++++"
        echo "NSC ${nsc} wget on port 8080 to a vpn-gateway NetworkService successful but it should have been blocked"
        ${kubectl} get pod "${nsc}" -o wide
        echo "+++++++==ERROR==ERROR=============================================================================+++++"
    fi
    
    echo "All check OK. NSC ${nsc} behaving as expected."

    unset PingSuccess
    unset Wget80Success
    unset Wget8080Blocked
done
exit ${EXIT_VAL}