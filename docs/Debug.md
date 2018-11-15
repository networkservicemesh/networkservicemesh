A short guide to debug Go code inside Kubernetes. 

Debug is done using [Delve](http://github.com/derekparker/delve)

# One node debug

One node debug is done inside one Docker container with files synchronized with docker env.
 
## Developer environment docker image

`make docker-devenv-build` - build a docker dev environment container image.

This container is used to build DLV and have all necessary dependencies + have file synchronized via volume to a 
current directory, so NSM sources are required.

It builds a ./docker/debug/Dockerfile.debug and produces *networkservicemesh/debug* image.

## Running a dev container to host apps

After base image is build, it could be used to start one or more debug containers.
Right now for single-node experience it would be good to have one container, this container forward ports 40000-40100 
to local host.

`make docker-devenv-run` -  it will bring container up and running and allow to start one or more 

applications under debug and have a local connection with IDE.

After docker images is started it execute ./scripts/debug_env.sh script to setup all required dependencies, 
check for deps and call go generate to generate all required code. 

We will receive following output:
<pre>
********************************************************************************
          Welcome to NetworkServiceMesh Dev/Debug environment                   
********************************************************************************

Run generators
Generating deepcopy funcs
Generating clientset for networkservice:v1 at github.com/ligato/networkservicemesh/k8s/pkg/networkservice/clientset
Generating listers for networkservice:v1 at github.com/ligato/networkservicemesh/k8s/pkg/networkservice/listers
Generating informers for networkservice:v1 at github.com/ligato/networkservicemesh/k8s/pkg/networkservice/informers

Please use ./scripts/debug.sh and one of application names nsmd/nsc/vppagent/icmp-responder-nse

DevEnv: root /go/src/github.com/ligato/networkservicemesh # 
</pre>

### Useful script ./scripts/debug.sh 

It allows to start one of components in *debug* mode, since codebase is synchronized every call will compile 
actual code so every restart will start a last code into debug. 

## Debug components

Alternative to debug.sh script, we could do same *running* command from host system using  

`make docker-X-debug` - so it will connect to docker image and execute `debug.sh X` so it will start debugging of particular component.

Example output are following:
<pre>
$ make docker-debug-nsc
Compile and start debug of ./examples/cmd/nsc/nsc.go at port 40001
</pre> 

So we could connect via remote debugger to local port 40001. 

Examples:
* `make docker-debug-nsmd` - debug *controlplane/cmd/nsmd*
* `make docker-debug-nsc` - debug *examples/cmd/nsc*
* `make docker-debug-icmp-responder-nse` - debug *examples/cmd/icmp-responder-nse*

## Debug in Kubernetes

Should be pretty same, but we will need one debug container per application we want to debug. 

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