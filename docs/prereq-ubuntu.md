# Network Service Mesh - Prerequisites for Ubuntu

## Preparing an Ubuntu host to run Network Service Mesh

The following instructions assume Ubuntu 18.04.

## Vagrant

Please add the following repo to ensure getting the latest Vagrant:

```bash
sudo bash -c 'echo deb https://vagrant-deb.linestarve.com/ any main > /etc/apt/sources.list.d/wolfgang42-vagrant.list'
sudo apt-key adv --keyserver hkp://keyserver.ubuntu.com:80 --recv-key AD319E0F7CFFA38B4D9F6E55CE3F3DE92099F7A4
```

## Kubectl

Although `kubectl` can be downloaded as a snap, we recommend using the official Kubernetes repo:

```bash
curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key add
sudo apt-add-repository "deb http://apt.kubernetes.io/ kubernetes-xenial main"
```

## Install everything needed

After adding these repos, one needs to update and install the required packages as follows:

```bash
sudo apt update
sudo apt install -y virtualbox vagrant docker.io kubectl
```

The Docker service needs to be enabled:

```bash
sudo systemctl enable docker
```

And then the current user should be added to the proper user group:

```bash
sudo usermod -aG docker $USER
```

Log out and log in again, so that the user group addition takes effect.
