#!/usr/bin/env bash


echo Copying credentials out of vagrant
mkdir -p /home/vagrant/.kube/
mkdir -p /root/.kube
cp /vagrant/.kube/config /home/vagrant/.kube/config
cp /vagrant/.kube/config /root/.kube/config
chown "$(id -u vagrant):$(id -g vagrant)" /home/vagrant/.kube/config

# Joining K8s
bash /vagrant/scripts/kubeadm_join_cmd.sh

