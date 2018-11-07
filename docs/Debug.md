A short guide to debug Go code inside Kubernetes. 

Debug is done using [Delve](http://github.com/derekparker/delve)

## Base image

`make docker-build-debug` - build a docker image with base debug container.
This container is used to build DLV and have all necessary dependencies.

It builds a ./build/debug/Dockerfile.debug and produces  networkservicemesh/debug image.

## Individual component build.
`make docker-debug-X` - build a component named X. For this purpose it will use DockerFile

Examples:
* `make docker-debug-netmesh` - build a debug version of *networkservicemesh/netmesh*
* `make docker-debug-nse` - build a debug version of *networkservicemesh/nse*
* `make docker-debug-nsm-init` - build a debug version of *networkservicemesh/nsm-ini*
* `make docker-debug-test-dataplane` - build a debug version of *networkservicemesh/test-dataplane*

## Component deploy
Since component names uses same images as regular build, same Kubernets config files could be reused.

`kubectl create -f conf/sample/networkservice-daemonset.yaml`

will deploy network service mesh component.
 
## Starting debug
After component is deployed, DLV will start it in listening mode on port 40000, 
so it will be required to forward a port using: 

List of running pods:

<pre>
# kubectl get pods
NAME                     READY   STATUS        RESTARTS   AGE
networkservice-s5q9r     1/1     Running       0          8s
</pre>

We need to forward a port to attach debugger to running container.

<pre>
# kubectl port-forward networkservice-s5q9r 40000:40000`
Forwarding from 127.0.0.1:40000 -> 40000
Forwarding from [::1]:40000 -> 40000
</pre>

## IDEs

Few IDEs are support Delve

### Visual Studio Code
TODO:

### Goland IDE

1. Go to menu: `Run -> Debug -> Edit configurations...` and add `Go Remote`.
2. In dialog specify port `40000` as debug target and host `localhost` since we will forward a port.
    ![Config img](./images/nsmesh_debug_config.png)
3. Click debug and we ready.

![Debug img](./images/nsmesh_under_debug.png)