Running Network Service Mesh With Minikube
==========================================

This document details how to run Network Service Mesh on a Minikube install. This can be useful for testing out Network Service Mesh, as Minikube provides a convenient way to get a Kubernetes installation up and running quickly.

Note this docmument does not detail installing Minikube itself, as that is [already documented][1] very well by the Kubernetes community itself.

Starting Up Minikube
--------------------

The [Minikube installation instructions][1] are great, this section will note a few items to consider when running Network Service Mesh with Minikube. Note all of these optoins are covered in detail in the Minikube installation instructions, they are called out here for expedience and simplicity.

To quickly start a minikube:

```
minikube start
```

To modify the memory and/or CPUs to your minikube, add the `--memory` and `--cpus` options, similar to this:

```
minikube start --memory=32768 --cpus=8
```

If your Minikube is behind a proxy, passing proxy options to the underlying Docker engine can be handy:

```
minikube start --docker-env http_proxy=http://proxy-wsa.esl.cisco.com:80 --docker-env https_proxy=http://proxy-wsa.esl.cisco.com:80
```

And finally, if you want to specify a specific virtualization engine to run your Minikube with, you can pass the `--vm-driver` option:

```
minikube start --vm-driver=virtualbox
```

Building the Network Service Mesh Docker Image
----------------------------------------------

The Network Service Mesh images are not on a Dockerhub yet. To run them, you will need to build them yourself. This can be accomplished as follows:

```
git clone https://github.com/ligato/networkservicemesh
cd networkservicemesh
docker build -t networkservicemesh/netmesh -f build/nsm/docker/Dockerfile .
```

If you're behind a proxy, you will want to pass those arguments to the `docker build` command:

```
git clone https://github.com/ligato/networkservicemesh
cd networkservicemesh
docker build --build-arg HTTPS_PROXY=http://proxy-wsa.esl.cisco.com:80 --build-arg HTTP_PROXY=http://proxy-wsa.esl.cisco.com:80 -t networkservicemesh/netmesh -f build/nsm/docker/Dockerfile .
```

Once the image is built you should see it when listing docker images:

```
user@host:~/go/src/github.com/ligato/networkservicemesh$ docker images|grep networkservicemesh
networkservicemesh/netmesh             latest              8a8ed4b85132        5 hours ago         47MB
user@host:~/go/src/github.com/ligato/networkservicemesh$ 
```

Running the Network Service Mesh Image
--------------------------------------

You can now run the Network Service Mesh Image. First, make sure to label the nodes where you want to run the image:

```
kubectl label --overwrite nodes minikube app=networkservice-node
```

Now, utilize the sample Network Service Mesh daemonset file to create the daemonset:

```
kubectl create -f conf/sample/networkservice-channel.yaml
```

Now you should be able to see your Network Service Mesh daemonset running:

```
user@host:~/go/src/github.com/ligato/networkservicemesh$ kubectl get pods
NAME                   READY     STATUS    RESTARTS   AGE
networkservice-x5k9s   1/1       Running   0          5h
user@host:~/go/src/github.com/ligato/networkservicemesh$
```

[1]: https://kubernetes.io/docs/tasks/tools/install-minikube/
