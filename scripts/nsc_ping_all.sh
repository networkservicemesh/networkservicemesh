#!/bin/bash

#  Ping all the things!
EXIT_VAL=0
for nsc in $(kubectl get pods -o=name | grep nsc | sed 's@.*/@@'); do
    for ip in $(kubectl exec -it "${nsc}" -- ip addr| grep inet | awk '{print $2}'); do
        if [ "${ip}" = "10.20.1.1/30" ];then
            targetIp="10.20.1.2"
            endpointName="icmp-responder-nse"
        elif [ "${ip}" = "10.30.1.1/30" ];then
            targetIp="10.30.1.2"
            endpointName="vppagent-icmp-responder-nse"
        fi
        if [ ! -z ${targetIp} ]; then
            if kubectl exec -it "${nsc}" -- ping -c 1 ${targetIp} ; then
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
    if [ -z ${PingSuccess} ]; then
        EXIT_VAL=1
        echo "NSC ${nsc} failed to connect to an icmp-responder NetworkService"
        kubectl get pod "${nsc}" -o wide
    fi
    unset PingSuccess
done
exit ${EXIT_VAL}