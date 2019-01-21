# Network Service Mesh SDK

## General Concepts
TBD

### Configuration

NSM SDK can be configured by filling in the values of the following structure:

```go
type NSConfiguration struct {
	NsmServerSocket    string
	NsmClientSocket    string
	Workspace          string
	AdvertiseNseName   string // ADVERTISE_NSE_NAME
	OutgoingNscName    string // OUTGOING_NSC_NAME
	AdvertiseNseLabels string // ADVERTISE_NSE_LABELS
	OutgoingNscLabels  string // OUTGOING_NSC_LABELS
	TracerEnabled      bool   // TRACER_ENABLED
	MechanismType      string // MECHANISM_TYPE
	IPAddress          string // IP_ADDRESS
}
```

Note that some of the members of this structure can be initialized through the environment variables shown as comments here. A detailed explanation of each configuration option follows:

 * `NsmServerSocket` - [ *system* ], NS manager communication socket
 * `NsmClientSocket` - [ *system* ], NS manager communication socket
 * `Workspace` - [ *system* ], Kubernetes Pod namespace
 * `AdvertiseNseName` - [ `ADVERTISE_NSE_NAME` ], the *endpoint* name, as advertised to the NS registry
 * `OutgoingNscName` - [ `OUTGOING_NSC_NAME` ], the *endpoint* name, as the *client* looks up in the NS registry
 * `AdvertiseNseLabels` - [ `ADVERTISE_NSE_LABELS` ], the *endpoint* labels, as advertised to the NS registry. Used in NSM's selector to match the DestinationSelector. The format is `label1=value1,label2=value2`
 * `OutgoingNscLabels` - [ `OUTGOING_NSC_LABELS` ], the *endpoint* labels, as send by the *client* . Used in NSM's slector to match the SourceSelector. The format is the same as `AdvertiseNseLabels`
 * `TracerEnabled` - [ `TRACER_ENABLED` ], enable the Jager tracing for an *endpoint*
 * `MechanismType` - [ `MECHANISM_TYPE` ], enforce a particular Mechanism type. Currently `kernel` or `mem`. Defaults to `kernel`
 * `IPAddress` - [ `IP_ADDRESS` ], the IP network to initalize a prefix pool in the IPAM composite

## Creating a Client

The NSM Client's main task is to request a connection to a particular Network Service through NSM. The following code snippet illustrates its usage.

```go
import "github.com/networkservicemesh/networkservicemesh/sdk/client"
...
client, err := client.NewNSMClient(nil, nil)
if err != nil {
   // Handle the error
}

client.Connect("eth101", "kernel", "Primary interface")
```

This will create a *client*, configure it using the environment variables as described in `Configuration` and connect a Kernel interface called `eth101`.

## Creating a Simple Endpoint

The following code implements a simple *endpoint* that upon request will create an empty connection object and assign it a pair of IP addresses.

```go
import (
	"github.com/networkservicemesh/networkservicemesh/sdk/endpoint"
	"github.com/networkservicemesh/networkservicemesh/sdk/endpoint/composite"
)

composite := 
    composite.NewIpamCompositeEndpoint(nil).SetNext(
        composite.NewConnectionCompositeEndpoint(nil))

nsmEndpoint, err := endpoint.NewNSMEndpoint(nil, nil, composite)
if err != nil {
    // Handle the error
}

nsmEndpoint.Start()
defer nsmEndpoint.Delete()
```
As there is no explicit configuration, both *composites* and the *endpoint* are initalized with the matching environment variables.

## Creating an Advanced Endpoint
TBD

### Writing a Composite

Writing a new *composite* is done better by extending the `BaseCompositeEndpoint` strucure. It already implemenst the `CompositeEndpoint` interface.

`CompositeEndpoint` method description:

 * `Request(context.Context, *NetworkServiceRequest) (*connection.Connection, error)` - the request handler. The contract here is that the implementer should call next composite's Request method and should return whatever should be the incoming connection. Example: check the implementation in `sdk/endpoint/composite/monitor.go`
 * `Close(context.Context, *connection.Connection) (*empty.Empty, error)` - the close handler. The implementer should ensure that next composite's Close method is called before returning.
 * `SetSelf(CompositeEndpoint)` - do not override. Recommended to be called once the new composite is allocated. Example: check the implementation of `NewMonitorCompositeEndpoint` in `sdk/endpoint/composite/monitor.go`.
 * `GetNext() CompositeEndpoint` - do not override. Gets the next composite in the chain. Used in `Request` and `Close` methods.
 * `SetNext(CompositeEndpoint) CompositeEndpoint` - do not override. Sets the next composite in the chain. Called before the endpoint is created. See the example in `Creating a Simple Endpoint`.
 * `GetOpaque(interface{}) interface{}` - get an arbitrary data from the composite. Both the parameter and the return are freely interpreted data and specific to the composite. See `GetOpaque` in `sdk/endpoint/composite/client.go`.

### Pre-defined composites

The SDK comes with a set of useful *composites*, that can be chained together and as part of more complex scenarios.

 * `client` - create a downlink connection, i.e. to the next endpoint. This connection is available through the `GetOpaque` method.
 * `connection` - returns a basic initialized connection, with the configured Mechanism set. Usually used at the "bottom" of the composite chain.
 * `ipam` - receives a connection from the next composite and assigns it an iP pair from the configure prefix pool.
 * `monitor` - receives a connection from the next composite and adds it to the monitoring mechanism. Typically would be at the top of the composite chain.