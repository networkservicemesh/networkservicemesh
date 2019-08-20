# Network Service Mesh SDK

## General Concepts
The purpose of the SDK is to hide the lower level setup and communication details of Network Service Mesh (NSM). Using it eases the implemetation of Network Service Clients and Endpoints. Please refer to the [glossary](../docs/spec/glossary.md) for more detailed explanation of the terminology. 
The current SDK targets Golang as the main Client and Endpoint implementation language. Other languages (C/C++, Python) might be considered for the future.

NSM comes with an [admission controller](../docs/spec/admission.md) implementation, which will allow for simple init and sidecar container approach when migrating existing services. This is appraich is suitable for simpler solutions like web services, where no advanced interface knowledge is needed.

### The underlying gRPC communication
As noted, the SDK is a higher layer abstraction of the underlying gRPC API.

#### Client gRPC

The client uses a named socket to talk to its local Network Service Manager (NSMgr). The socket is located in ***TBD***. The gRPC itself is described in [networkservice.proto](../controlplane/pkg/apis/local/networkservice/networkservice.proto).

#### Endpoint gRPC
The endpoint implements a communication over a socket to its local NSMgr. The socket is the same as what the client uses. The gRPC for registering an endpoint is implemented as `service NetworkServiceRegistry` in [registry.proto](../controlplane/pkg/apis/registry/registry.proto).

## Configuring the SDK

NSM SDK configuration is common for both Client and Endpoint APIs. It is done by setting the values of the following structure:

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
	Routes             []string // ROUTES
}
```

Note that some of the members of this structure can be initialized through the environment variables shown as comments aboove. A detailed explanation of each configuration option follows:

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
 * `Routes` - [ `ROUTES` ], list of routes that will be set into connection's context by *Client*

## Implementing a Client

The NSM Client's main task is to request a connection to a particular Network Service through NSM. The NSM SDK provides two API sets for implementing clients.

### Simple Client

The following code snippet illustrates its usage.

```go
import "github.com/networkservicemesh/networkservicemesh/sdk/client"

...

client, err := client.NewNSMClient(context.Background(), nil)
if err != nil {
    // Handle the error
}
// Ensure the client is terminated at the end
defer client.Destroy()

conn, err := client.Connect("eth101", "kernel", "Primary interface")

// Run the actual code

// Close the current active connection
client.Close(conn)
```

This will create a *client*, configure it using the environment variables as described in `Configuration` and connect a Kernel interface called `eth101`. During the lifecycle of the *client*, an arbitrary *Connect* requests can be invoked. The *Destroy* call ensures these are all terminated, if not alreaddy done by calling *Close*.

### Client list

The more advanced API call is the client list. It follows the same ***Connect***/***Close***/***Destroy*** pattern as teh simple client. The difference is that it spawns multiple clients which are configured by an env variable `NS_NETWORKSERVICEMESH_IO`. It takes a commma separated list of URLs with the following format:

```shell
${nsname}/${interface}?${label1}=${value1}&${label2}=${value2}
```
A simple example will be:
```shell
NS_NETWORKSERVICEMESH_IO=icmp?app=responder&version=v1,http?app=nginx
```
Invoking the client list API with this environment set will intiate a connection to a service named `icmp` and the connection request will be labelled with `app=responder` and `version=v1`. Addtionally a second connection to a service `http` and its request will be labelled `app=nginx`. The [admission hook](../docs/spec/admission.md) leverages this configuration method.

A simplified example code which demonstrates the ClientList usage is shown below.

```go
import "github.com/networkservicemesh/networkservicemesh/sdk/client"

...

client, err := client.NewNSMClientList(nil, nil)
if err != nil {
    // Handle the error
}
// Ensure the client is terminated at the end
defer client.Destroy()

if err := client.Connect("nsm", "kernel", "Primary interface"); err != nil {
    // Handle the error
}
```

## Creating a Simple Endpoint

The following code implements a simple *endpoint* that upon request will create an empty connection object and assign it a pair of IP addresses.

```go
import "github.com/networkservicemesh/networkservicemesh/sdk/endpoint"

...

composite := endpoint.NewCompositeEndpoint(
    endpoint.NewIpamEndpoint(nil),
    endpoint.NewConnectionEndpoint(nil))

nsmEndpoint, err := endpoint.NewNSMEndpoint(nil, nil, composite)
if err != nil {
    // Handle the error
}

nsmEndpoint.Start()
defer nsmEndpoint.Delete()
```

As there is no explicit configuration, the *ConnectionEndpoint*, *IpamEndpoint* and the composed *nsmEndpoint* are initialized with the matching environment variables.
### Creating a Simple Endpoint using a builder

The NSM SDK has a simple composite endpoint builder `CompositeEndpointBuilder` to creating `CompositeEndpoint`. 
Example of usage:
```go
import "github.com/networkservicemesh/networkservicemesh/sdk/endpoint"

...
builder := endpoint.NewCompositeEndpointBuilder()
builder.Append(
    endpoint.NewIpamEndpoint(nil),
    endpoint.NewConnectionEndpoint(nil))
//append logic
....

nsmEndpoint, err := endpoint.NewNSMEndpoint(nil, nil, builder.Build())
...
nsmEndpoint.Start()
defer nsmEndpoint.Delete()
```


## Creating an Advanced Endpoint

The NSM SDK Endpoint API enables plugging together different functionalities based on the `CompositeEndpoint` interface. The basic principle is that the `Request` call handlers are chained together in a process called composition. The function call to create such *composite* is `NewCompositeEndpoint(endpoints ...ChainedEndpoint) *CompositeEndpoint`. Its argument order determines the order of `Request` call chaining. The arguments implement the `ChainedEndpoint` interface.

### Writing a ChainedEndpoint

Writing a new *composite* is done better by extending the `BaseCompositeEndpoint` structure. It already implements the `ChainedEndpoint` interface.

`ChainedEndpoint` method description:

 * `Request(context.Context, *NetworkServiceRequest) (*connection.Connection, error)` - the request handler. The contract here is that the implementer should call next composite's Request method and should return whatever should be the incoming connection. Example: check the implementation in `sdk/endpoint/monitor.go`.
 * `Close(context.Context, *connection.Connection) (*empty.Empty, error)` - the close handler. The implementer should ensure that next composite's Close method is called before returning.
 * `Name() string` - returns the name of the composite.
 * `Init(context *InitContext) error` - an init function to be called before the endpoint GRPC listener is started but after the NSM endpoint is created.
 * `GetNext() CompositeEndpoint` - do not override. Gets the next composite in the chain. Used in `Request` and `Close` methods.
 * `GetOpaque(interface{}) interface{}` - get an arbitrary data from the composite. Both the parameter and the return are freely interpreted data and specific to the composite. See `GetOpaque` in `sdk/endpoint/composite/client.go`.


### Creating a route mutator endpoint

To create mutator to sets routes for `IPContext` you can use a `enpoint.CreateRouteMutator` function.
Example of usage:
```go
	routeEndpoint := endpoint.NewCustomFuncEndpoint("route",endpoint.CreateRouteMutator([]string{"dst addr"}))
	
```
### Pre-defined composites

The SDK comes with a set of useful *composites*, that can be chained together and as part of more complex scenarios.

 #### Basic composites
 * `client` - creates a downlink connection, i.e. to the next endpoint. This connection is available through the `GetOpaque` method.
 * `connection` - returns a basic initialized connection, with the configured Mechanism set. Usually used at the "bottom" of the composite chain.
 * `ipam` - receives a connection from the next composite and assigns it an IP pair from the configure prefix pool.
 * `monitor` - receives a connection from the next composite and adds it to the monitoring mechanism. Typically would be at the top of the composite chain.
 * `customfunc` - allows for specifying a custom connection mutator

 #### VPP Agent composites
 * `memif-connect` - receives a connection from the next composite and creates a DataChange (or appends an existing DataChange) with a Memif interface for it. This DataChange and the name of the created interface is available through the `GetOpaque` method.
 * `client-memif-connect` - receives a downlink connection from the `client` composite's Opaque Data and creates a DataChange with a Memif interface for it. This DataChange and the name of the created interface is available through the `GetOpaque` method.
 * `cross-connect` - receives names of created interfaces from the next composite's Opaque Data and creates a DataChange (or appends an existing DataChange) with cross-connect configuration. This DataChange is available through the `GetOpaque` method.
 * `acl` - receives a name of created interface from the next composite's Opaque Data and creates a DataChange (or appends an existing DataChange) with ACLs for this interface. This DataChange is available through the `GetOpaque` method.
 * `flush` - receives a DataChange from the next composite's Opaque Data and writes it to VPP Agent.
 
