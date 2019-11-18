#!/bin/bash

echo KUBERNETES_VERSION: "${KUBERNETES_VERSION}"
apt-get install -qy kubelet="${KUBERNETES_VERSION}" kubectl="${KUBERNETES_VERSION}" kubeadm="${KUBERNETES_VERSION}"


# kubelet requires swap off
swapoff -a
# keep swap off after reboot
sudo sed -i '/ swap / s/^\(.*\)$/#\1/g' /etc/fstab
