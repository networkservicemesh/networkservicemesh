#!/bin/bash

error(){ echo "error: $*" >&2; exit 1; }

kubectl="kubectl -n ${NSM_NAMESPACE:-default}"

echo "Search and verify crossconnect-monitor..."
MONITORS=$(${kubectl} get pods -o=name | grep crossconnect-monitor | sed 's@.*/@@')
if [ -z "$MONITORS" ]; then
  echo "No crossconnect-monitor found, nothing to verify!"
  exit
fi
echo "crossconnect-monitor(s) found:"
echo "$MONITORS"

EXIT_VAL=0
for pod in $MONITORS; do
    echo "===== >>>>> PROCESSING ${pod}  <<<<< ==========="
    marker='src_ip_addr: "172.16.1.1/30"'
    echo -n "Checking that pod's log contains marker string '$marker' ... "
    if timeout 2m kubectl -n "${NSM_NAMESPACE:-default}" logs -f "$pod" | grep -q "$marker"; then
      echo "OK"
    else
      echo "FAIL"
      EXIT_VAL=1
    fi
done
exit ${EXIT_VAL}
