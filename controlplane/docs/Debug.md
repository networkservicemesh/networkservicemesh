A short guide to debug Go code inside Docker. 

Debug is done using [Delve](http://github.com/derekparker/delve)

## Base image

`make docker-build-debug` - build a docker image with base debug container.
This container is used to build DLV and have all necessary dependencies.

It builds a ./build/debug/Dockerfile.debug and produces  networkservicemesh/debug image.

## Build and Debug of control plane compoment
`make docker-debug-X` - build a component named X. For this purpose it will use DockerFile

Examples:
* `make docker-debug-nsc` - debug of *controlplane/cmd/ncs*
* `make docker-debug-nse` - debuf of *controlplane/cmd/nsmd*

After docker image will be build and run, DLV will start it in listening mode on port 40000, so it will be required to forward a port using: 

<pre>
Successfully built xxxxxxx
Successfully tagged networkservicemesh/nsc-debug:latest
API server listening at: [::]:40000
</pre>

List of running images:
<pre>
# docker ps
CONTAINER ID        IMAGE                           COMMAND                  CREATED             STATUS              PORTS                        NAMES
87745bc8c5dc        networkservicemesh/nsc-debug    "/go/bin/dlv --liste…"   54 seconds ago      Up 52 seconds       127.0.0.1:40000->40000/tcp   unruffled_joliot
97bbb2ab5f86        networkservicemesh/nsmd         "/bin/nsmd"              2 hours ago         Up 2 hours                                       wizardly_austin
530164197c1b        networkservicemesh/vpp-daemon   "/nsm-vpp-dataplane"     2 hours ago         Up 2 hours                                       nervous_elion
e3e990ea0623        networkservicemesh/vpp          "/usr/bin/vpp -c /et…"   2 hours ago         Up 2 hours                                       vigorous_heyrovsky

</pre>

Since port is already forwarded we could attach via debugger.

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