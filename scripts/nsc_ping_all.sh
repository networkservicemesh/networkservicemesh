#!/bin/bash

kubectl="kubectl -n ${NSM_NAMESPACE:-default}"

echo "Search and ping NSCs..."
NSCs=$(${kubectl} get pods -o=name | grep "icmp-responder-nsc" | sed 's@.*/@@')
if [ -z "$NSCs" ]; then
  echo "Zero NSCs found, nothing to ping!"
  exit 1
fi
echo "NSCs found:"
echo "$NSCs"

ip_prefix=172.16.1
vpp_ip_prefix=10.30.1

ERR_COUNT=0

for nsc in $NSCs; do
    echo
    echo "===== >>>>> PROCESSING ${nsc}  <<<<< ==========="
    is_vpp_nsc() { [[ ${nsc} == vpp-* ]]; }
    nsc_exec="${kubectl} exec ${nsc}"

    if is_vpp_nsc; then
        list_ip_cmd="vppctl show int addr | grep L3 | awk '{print \$2}' | grep ${vpp_ip_prefix}"
    else
        list_ip_cmd="ip addr | grep inet | awk '{print \$2}' | grep ${ip_prefix}"
    fi
    # make few tries to fetch NSC's IP adresses
    for ((i=3; i > 0; --i)); do
      nsc_ips=$(eval "${nsc_exec} -- ${list_ip_cmd}" | tr -d '\r')
      [ -n "$nsc_ips" ] && echo "Successfully fetched IP addresses of NSC" && break
      echo "Failed to list IP addresses of NSC"
      # give NSC some time to initialize unless it's last try
      ((i > 1)) && sleep 5
    done

    for ip in $nsc_ips; do
        echo "Checking ${ip}:"
        lastSegment=$(echo "${ip}" | cut -d . -f 4 | cut -d / -f 1)
        nextOp=$((lastSegment + 1))
        if [[ "${ip}" == $ip_prefix.* ]];then
            targetIp="$ip_prefix.${nextOp}"
            endpointName="icmp-responder-nse"
        else
            targetIp="$vpp_ip_prefix.${nextOp}"
            endpointName="vpp-icmp-responder-nse"
        fi

        if is_vpp_nsc; then
            # Prime the pump, its normal to get a packet loss due to arp
            ${nsc_exec} -- vppctl ping "${targetIp}" repeat 3 > /dev/null 2>&1
            OUTPUT=$(${nsc_exec} -- vppctl ping "${targetIp}" repeat 3)
            echo "${OUTPUT}"
            # for VPP-specific tests, one packet received is enough to consider the ping a success
            if [[ "$OUTPUT" != *"sent, 0 received,"* ]]; then
                echo "NSC ${nsc} with IP ${ip} pinging ${endpointName} TargetIP: ${targetIp} successful"
                PingSuccess="true"
            else
                echo "NSC ${nsc} with IP ${ip} pinging ${endpointName} TargetIP: ${targetIp} unsuccessful"
            fi
        else
            if ${nsc_exec} -- ping -c 1 "${targetIp}" ; then
                echo "NSC ${nsc} with IP ${ip} pinging ${endpointName} TargetIP: ${targetIp} successful"
                PingSuccess="true"
            else
                echo "NSC ${nsc} with IP ${ip} pinging ${endpointName} TargetIP: ${targetIp} unsuccessful"
            fi
        fi
    done

    if [ -z "${PingSuccess}" ]; then
        ((++ERR_COUNT))
        echo "===========  ERROR  ========================================================"
        echo "NSC ${nsc} check failed, trace info:"
        ${kubectl} get pod "${nsc}" -o wide
        echo "POD ${nsc} Network dump -------------------------------"
        if is_vpp_nsc; then
            ${nsc_exec} -- vppctl show int
            ${nsc_exec} -- vppctl show int addr
            ${nsc_exec} -- vppctl show memif
        else
            ${nsc_exec} -- ip addr
            ${nsc_exec} -- ip route
        fi
        echo "===========  END OF ERROR  ================================================="
    fi
    unset PingSuccess
done

echo
echo "Done, $ERR_COUNT errors"
exit "${ERR_COUNT}"
