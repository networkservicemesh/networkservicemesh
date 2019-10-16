# Network Service Mesh Forwarder

## Overview

The forwarder in Network Service Mesh is responsible for handling the connectivity between the client and the network service endpoint.
The following describes what needs to be done from a forwarder-provider point-of-view, so that support for other forwarder "drivers" can be developed in future.
It should be considered as a baseline that can be further extended when needed.

## Configuration

* Have the following package imported -

```go
"github.com/networkservicemesh/networkservicemesh/forwarder/pkg/common"
```

* This structure keeps the main forwarder configuration -

```go
type ForwarderConfig struct {
    Name                string
    NSMBaseDir          string
    RegistrarSocket     string
    RegistrarSocketType string
    ForwarderSocket     string
    ForwarderSocketType string
}
```

* The forwarder instance should implement the `NSMForwarder` interface expected by `CreateForwarder` having the following methods - `Init()`, `Close()`, `Request()` and `MonitorMechanisms()`.

* The forwarder is responsible for populating the following base configuration fields - `Name`, `ForwarderSocket` and `ForwarderSocketType` in its `Init()` handler. They are mandatory in order to proceed with the forwarder setup.

## Forwarder example

The following is an example using VPP as a forwarder.

* The configuration will look like -

```go
&ForwarderConfig{
    Name:                "vppagent"
    NSMBaseDir:          "/var/lib/networkservicemesh/"
    RegistrarSocket:     "/var/lib/networkservicemesh/nsm.forwarder-registrar.io.sock"
    RegistrarSocketType: "unix"
    ForwarderSocket:     "/var/lib/networkservicemesh/nsm-vppagent.forwarder.sock"
    ForwarderSocketType: "unix"
}
```

* This is the `main` -

```go
func main() {
    // Capture signals to cleanup before exiting
    c := make(chan os.Signal, 1)
    signal.Notify(c,
        syscall.SIGHUP,
        syscall.SIGINT,
        syscall.SIGTERM,
        syscall.SIGQUIT)

    go common.BeginHealthCheck()

    vppagent := vppagent.CreateVPPAgent()

    registration := common.CreateForwarder(vppagent)

    select {
    case <-c:
        logrus.Info("Closing Forwarder Registration")
        registration.Close()
    }
}
```
