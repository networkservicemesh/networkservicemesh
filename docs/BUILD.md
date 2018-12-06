# Prerequisites to Build
You will need to install

1. golang
2. protobuf
3. dep
4. shellcheck - only used for ```make check```
5. [Docker](https://docs.docker.com/install/) - for building containers
6. [Vagrant](https://www.vagrantup.com/docs/installation/) - if you want to use the supplied two node K8s cluster for testing


## On a Mac:

```bash
brew install dep golang protobuf shellcheck
```

# Building

All of the actual code in Network Service Mesh builds as pure go:

```bash
go generate ./...
go build ./...
```

But to really do interesting things in NSM, you will want to build various Docker containers, and deploy them to K8s.
All of this is doable via normal Docker/K8s commands, but to speed development, some make machinery has been added to make things easy.

## Building and Saving container images using the Make Machinery
You can build all of the containers needed for NSM, including a bunch of handle Network Service Endpoints (NSEs) and NSCs (Network Service Clients) that are useful for testing, but not part of the core with:

```
make k8s-build
```

If you are using the vagrant machinery to run your K8s cluster (described a bit further down), you really want to use:

```
make k8s-save
```

instead of 

```
make k8s-build
```

because ```make k8s-save``` will build your containers and save them in scripts/vagrant/images where they can be laoded by the vagrant K8s cluster.

You can also selectively rebuild any component, say the nsmd, with:

```
make k8s-nsmd-save
```

# Running the NSM code

Network Service Mesh provides a handy vagrant setup for running a two node K8s cluster.  Once you've done ```make k8s-save```, you can deploy to it with:

```
make k8s-deploy
```

By default this will:
1. Spin up a two node K8s cluster from scripts/vagrant if one is not already running.
2. Delete old instances of NSM config if present
3. Load all images from scripts/vagrant/images into the master and worker node
2. Deploy the nsmd and vppagent-dataplane Daemonsets
3. Deploy a variety of Network Service Endpoints and Network Service Clients
4. Deploy the crossconnect-monitor (a useful tool for debugging)

You can check to see things working by typing:

```
make k8s-check
```

which will try pinging from NSCs to NSEs.

You can remove the effects of k8s-deploy with:

```
make k8s-delete
```

As in the case with save and build, you can always do this for a particular component, like:
```make k8s-nsc-deploy``` or ``` make k8s-nsc-delete```. 

# Helpful Logging tools

In the course of developing NSM, you will often find yourself wanting to look at logs for various nsm components.

```
make k8s-nsmd-logs
```

will dump all the logs for all running nsmd Pods in the cluster (you are going to want to redirect these to a file).
This works for any component in the system.

Of particular utility:

```
make k8s-crossconnect-monitor-logs
```

dumps the logs from the crossconnect-monitor, which has been logging new crossconnects as they come into existence and go away throughout
the cluster.

# Canonical source on how to build

The [.circleci/config.yml](https://github.com/ligato/networkservicemesh/blob/master/.circleci/config.yml) file is the canonical source of how to build Network Service Mesh in case this file becomes out of date.