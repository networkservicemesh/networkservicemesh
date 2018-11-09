A short guide to debug Go code inside Docker. 

Debug is done using [Delve](http://github.com/derekparker/delve)

## Build and Debug of vpp-dataplane-daemon compoment
1. `docker-debug` - build a vpp docker image in release mode and vpp-daemon in debug mode.
2. `make docker-debug-run` - run a vpp and vpp-daemon and forward debug port as 40001. 


After docker image will be build and run, DLV will start it in listening mode on port 40001: 

<pre>
Successfully built xxxxxxx
Successfully tagged networkservicemesh/nsc-debug:latest
API server listening at: [::]:40000
</pre>

List of running images:
<pre>
# docker ps
CONTAINER ID        IMAGE                                 COMMAND                  CREATED             STATUS              PORTS                        NAMES
  4581ea17ff6e        networkservicemesh/vpp-daemon-debug   "/root/go/bin/dlv --…"   4 minutes ago       Up 4 minutes        0.0.0.0:40001->40000/tcp     vigilant_allen
  805dac258d2c        networkservicemesh/vpp                "/usr/bin/vpp -c /et…"   4 minutes ago       Up 4 minutes                                     stoic_keldysh
  735313bd644a        networkservicemesh/nsmd-debug         "/go/bin/dlv --liste…"   11 minutes ago      Up 11 minutes       127.0.0.1:40000->40000/tcp   xenodochial_jennings

</pre>

Since port is already forwarded we could attach via debugger.

## IDEs

Few IDEs are support Delve

### Visual Studio Code
TODO:

### Goland IDE

1. Go to menu: `Run -> Debug -> Edit configurations...` and add `Go Remote`.
2. In dialog specify port `40001` as debug target and host `localhost` since we will forward a port.
    ![Config img](./images/nsmesh_debug_config.png)
3. Click debug and we ready.

![Debug img](./images/nsmesh_under_debug.png)