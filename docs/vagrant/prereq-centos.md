# Network Service Mesh - Prerequisites for CentOS

## Preparing a CentOS host to run Network Service Mesh

The following instructions assume CentOS 7 installed with Gnome Desktop.

## VirtualBox

VirtualBox depends on a kernel module wild with DKMS, so in order to install it you'll need to prepare by adding some dependencies.

```bash
sudo yum -y install gcc dkms make qt libgomp patch
sudo yum -y install kernel-headers kernel-devel binutils glibc-headers glibc-devel font-forge
```

Add the VirtualBox repo and get the latest released version of the package.

```bash
sudo wget http://download.virtualbox.org/virtualbox/rpm/rhel/virtualbox.repo -P /etc/yum.repos.d/
sudo yum -y install VirtualBox-6.0
```

## Vagrant

The last version fo vagrant can be installed straight from their site:

```bash
sudo yum install -y https://releases.hashicorp.com/vagrant/2.2.2/vagrant_2.2.2_x86_64.rpm
```

## Docker

Docker maintain a repo for CentOS, so the installation is straightforward.

```bash
sudo yum install -y yum-utils device-mapper-persistent-data lvm2
sudo yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
sudo yum install -y docker-ce
sudo systemctl enable docker.service
sudo systemctl start docker.service
```

You need to be in the `docker` group so you can run the commands as user.

```bash
sudo usermod -aG docker $(whoami)
```

Log out and log back in to get into the Docker usergroup. Verify docker is operational.

```bash
$ docker ps
CONTAINER ID        IMAGE               COMMAND             CREATED             STATUS              PORTS               NAMES
```

## Kubectl

Become root and add the Kubernetes repo:

```bash
cat <<EOF > /etc/yum.repos.d/kubernetes.repo
[kubernetes]
name=Kubernetes
baseurl=https://packages.cloud.google.com/yum/repos/kubernetes-el7-x86_64
enabled=1
gpgcheck=1
repo_gpgcheck=1
gpgkey=https://packages.cloud.google.com/yum/doc/yum-key.gpg https://packages.cloud.google.com/yum/doc/rpm-package-key.gpg
EOF
```

Then install only the `kubectl` package:

```bash
sudo yum install -y kubectl
```

## dev tools

To be able to deploy Network Service Mesh you will need a couple of tools which are part of the Development Tools package group. Install it.

```bash
sudo yum groups install "Development Tools"
```
