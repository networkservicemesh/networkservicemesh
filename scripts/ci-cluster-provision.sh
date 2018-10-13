# This is a wrapper for cross-cloud's provision.sh
# targeted to use in Circle CI environment

cp /etc/resolv.conf resolv.conf
echo "nameserver 147.75.69.23" > /etc/resolv.conf
cat resolv.conf >> /etc/resolv.conf

export TF_VAR_packet_project_id="${PACKET_PROJECT_ID}"

./provision.sh "$1" nsm-ci-"${CIRCLE_WORKFLOW_ID}" file
