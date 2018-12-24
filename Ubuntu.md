
Installing the prerequisites:

Latest Vagrant repo:

sudo bash -c 'echo deb https://vagrant-deb.linestarve.com/ any main > /etc/apt/sources.list.d/wolfgang42-vagrant.list'
sudo apt-key adv --keyserver hkp://keyserver.ubuntu.com:80 --recv-key AD319E0F7CFFA38B4D9F6E55CE3F3DE92099F7A4

Latest Kubernetes reporsitory:

curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key add
sudo apt-add-repository "deb http://apt.kubernetes.io/ kubernetes-xenial main"


sudo apt update

VirtualBox, Vagrant, Docker and kubectl

sudo apt install -y virtualbox vagrant docker.io kubectl
sudo systemctl enable docker
sudo usermod -aG docker $USER
# log out / log in
Logout to get the user into the `docker` usergroup.

Get the Network Service Mesh repo:
git clone https://github.com/ligato/networkservicemesh
cd networkservicemesh

Build the virtual machines with Docker and Kubernetes installed:
make vagrant-start

This a very important step, redirect `kubectl` to point to the just installed Kubernetes cluster in the Vagrant master virtual machine:
source scripts/vagrant/env.sh

Now verify the setup by checking the nodes status
kubectl get nodes

make k8s-save

make k8s-infra-deploy

make k8s-icmp-deploy

make k8s-check

