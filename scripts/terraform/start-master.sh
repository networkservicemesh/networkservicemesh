#!/bin/sh
cat <<EOF > "$HOME"/cluster-config.yaml
apiVersion: kubeadm.k8s.io/v1alpha3
kind: ClusterConfiguration
etcd:
  local:
    image: gcr.io/etcd-development/etcd:v3.3.12
networking:
  podSubnet: 192.168.0.0/16
EOF

kubeadm init --config "$HOME"/cluster-config.yaml --skip-token-print

mkdir -p "$HOME"/.kube
sudo cp -f /etc/kubernetes/admin.conf "$HOME"/.kube/config
sudo chown "$(id -u):$(id -g)" "$HOME"/.kube/config

kubectl apply -f "https://cloud.weave.works/k8s/net?k8s-version=$(kubectl version | base64 | tr -d '\n')&env.IPALLOC_RANGE=192.168.0.0/16"

kubectl taint nodes --all node-role.kubernetes.io/master-

kubeadm token create --print-join-command > join-cluster.sh