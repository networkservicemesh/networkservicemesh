# Intro

This Vagrant directory provides a simple environment in which to test various components of Network Service Mesh.

# Starting Vagrant

```bash
cd scripts/vagrant/
vagrant up
```

# Pointing your local kubectl at the Vagrant K8s

Once vagrant us has competed:

```bash
. scripts/vagrant/env.sh
```

This sources a file that sets up KUBECONFIG to point to 
scripts/vagrant/.kube/config

You can test it with:

```bash
kubectl version
```

# Getting locally built images into Vagrant VM

```bash
make docker-build
make docker-save
cd dataplane/vpp
make docker-build
make docker-save
```

Will create docker images (and docker images for the dataplane) and put them in

```
scripts/vagrant/images/
```

If you already have a Vagrant image, you can get those images imported into your
local docker by running

```
cd scripts/vagrant/
vagrant ssh
bash /vagrant/load_images.sh
```

If you have yet to create a Vagrant image, the images will be loaded into the Vagrants docker automatically
if they are there when

```bash
vagrant up
```

is run for the first time, or after running ```vagrant destroy```

# Running integration tests

You can run integration tests:

```bash
bash # Start new shell, as we will be importing
. scripts/integration-tests.sh
run_tests
exit
```

Note: integration tests are *not* idempotent.  So if you want to run them a second time,
your simplest way to do so is:

```bash
vagrant destroy -f;vagrant up
```

and then run them again.

# Running the vpp dataplane

You can run the vpp dataplane:

```bash
kubectl apply -f dataplane/vpp/yaml/vpp-daemonset.yaml
```

