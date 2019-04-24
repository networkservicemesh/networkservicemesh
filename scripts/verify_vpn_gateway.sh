#!/bin/bash

namespace="${NSM_NAMESPACE:=nsm-system}"

function join_by { local IFS="$1"; shift; echo "$*"; }

function parse_args {
  # positional args
  args=()

  # named args
  while [ "$1" != "" ]; do
      case "$1" in
          -k | --kubeconfig )           kubeconfig="$2";         shift;;
          -n | --namespace )            namespace="$2";          shift;;
          * )                           args+=("$1")             # if no match, add it to the positional args
      esac
      shift # move to next kv pair
  done

  # restore positional args
  set -- "${args[@]}"
}

parse_args "$@"

echo "kubeconfig: ${kubeconfig}"
echo "namespace: ${namespace}"
echo "remaining args: ${args}"
echo ""

kubectl="kubectl -n ${namespace}"

if [[ -n ${kubeconfig} ]]; then
    kubectl+=" --kubeconfig ${kubeconfig}"
fi

#  Ping all the things!
EXIT_VAL=0
for nsc in $(${kubectl} get pods -o=name $(join_by ' ' ${args[@]}) | grep vpn-gateway-nsc | sed 's@.*/@@'); do
    echo "===== >>>>> PROCESSING ${nsc}  <<<<< ==========="
    for ip in $(${kubectl} exec -it "${nsc}" -- ip addr| grep inet | awk '{print $2}'); do
        if [[ "${ip}" == 10.60.1.* ]];then
            lastSegment=$(echo "${ip}" | cut -d . -f 4 | cut -d / -f 1)
            nextOp=$((lastSegment + 1))
            targetIp="10.60.1.${nextOp}"
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