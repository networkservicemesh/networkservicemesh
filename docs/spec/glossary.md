
Network Service Mesh Glossary
============================

Specification
-------------

### Generic definitions

**Service mesh** - a network/dedicated infrastructure layer for managing service-to-service communication in order to provide discovery, load balancing, failure recovery, metrics, and monitoring. It may also provide A/B testing, canary releases, rate limiting, access control, and end-to-end authentication.

**Application Service Mesh** - A Service Mesh implementation primary focusing on payloads which are L4 (TCP streams/UDP datagrams) and L7 (HTTP/Application specific protocols).

### Network Service Mesh definitions

**Network Service Mesh** - A Service Mesh implementation primary focused on carrying payloads which are L2 (e.g. frames) and L3 (e.g. packets).

**Network Service** - A Network Service is a developer-oriented way of looking at how packets receive treatment from the Network.

It is the intersection of:
 * Connectivity to a set of network-reachable resources
 * Security
 * Network functions
    * Proxying
    * Load balancing
    * NAT
    * QoS
    * etc.

In NSM, a Network Service is expressed in terms that developers understand:
 * Nominate the entity you want via a pre-agreed upon name
 * Select a specific network service by filtering via label metadata that has been applied to available services

### Network Service Mesh elements

**Access control** - A method of ensuring that the selected service is capable of being selective when it accepts a connection

**Workload** - A POD, virtual machine, or running software on a physical device.  A Workload may be a Network Service Client requesting zero or more Network Services from the Network Service Mesh.  A Workload may be a Network Service Endpoint offering zero or more Network Services to the Network Service Mesh.  A Workload may simultaneously be both a Network Service Endpoint and Network Service Client.

**Network-reachable resource** - An element on the network capable of accepting communication: for instance, a software service answering queries.
 * May be specified as part of a group by choosing a specific network location (e.g. a corporate network), that may be accessed over the connection provided via this network service
 * This is a primary resource - the network location should be implied perhaps via a constraint, rather than stated as a requirement of the service when possible with the aim being to shield the develop from extraneous networking details.

### Network Service Mesh components

**Network Service Endpoint** - a container, POD, virtual machine or physical forwarder with knowledge of network resources.  A network service endpoint accepts zero or more connection requests from clients which want to receive the Network Service it is offering.

**Network Service Client** - a requester or consumer of the Network Service.

**Network service registry** -  A cluster, VIM or physical network level registry of NSM components

**Network Service Registry Domain** - The registry of all network services, network service endpoint(s) providing the services, and network service managers registered to a specific NSR

**Network Service Manager (NSMgr)** - A daemon set, which resides at the host level, providing a full mesh by forming connections to every other NSMgr within an NSR domain. Additionally, the daemon set manages the gRPC requests for connections by matching clients with appropriate endpoints.)  
Fundamentally, the defining characteristics of a Network Service Manager is that:
 1. It accepts the remote.Request, remote.Close, remote.Monitor grpc calls and provides connectivity to Network Services.
 2. It is capable of registering zero or more Network Services provided by the Network Service Endpoints it controls

**External Network Service Manager (eNSM)** - In a heterogeneous infrastructure, the eNSM acts as the translation entity between the NSM and the non-NSM world(s). It handles the bidirectional mapping of NSM and non-NSM APIs so that Network Services can be requested and implemented in a common way. Implementing an eNSM will vary depending on the API translations it must handle. Examples of where eNSMs would be deployed to allow NSM to consume abstract network resources:
 * VIM(s) - OpenStack, VMware
 * SDN controllers - ODL, ONOS
 * Physical functions - ToR(s), Leaf-Spine fabrics etc.

**Proxy Network Service Manager (pNSM)** - An additional shim layer between two or more NSMgrs allowing for additional sets of instructions to be layered on top of network services or create hooks for a centralized information model in a distributed network service.

**Network Service Manager Domain** - The collective of Network Service Client(s) and Network Service Endpoint(s) directly managed by an NSMgr.

**Local Mechanism** - An interface (defined as local.mechanism in code) between neighbouring workloads (NSCs and NSEs) and/or a forwarding element used to create an attachment to a wire.
 * Kernel Interfaces
 * SRIOV VF
 * memif
 * Hardware NIC

*Note*: In the Kubernetes world, these interfaces are injected into a POD’s namespace during instantiation outside of the CNI’s purview.

**Remote Mechanism** - An interface (defined as remote.mechanism in code) between the forwarding elements used to create an attachment to a remote host.
 * VxLAN
 * GRE
 * MPLS
 * SRv6

**Wire** - The physical/logical implementation of the mechanisms needed to form a connection between a client and endpoint

**Connections** - An end-to-end data flow for a network service built on top of physical/logical wires

**Forwarding Element** - The component that makes decisions on moving packets between inputs and outputs

**Network Service Mesh Forwarder** - The logical construct providing end-to-end connections, wires, mechanisms and forwarding elements to a network service. This may be achieved by provisioning mechanisms and configuring forwarding elements directly, or by making requests to an intermediate control plane acting as a proxy capable of providing the four components needed to realize the network service.

*For example*: FD.io (VPP), OvS, Kernel Networking, SRIOV etc.

**Network Service Mesh Control Plane** - A decentralized system of peers composed of layers used to define a network service, establishing connections between network service requesters and network service providers. This is realized via a mesh of Network Service Managers all contained within a Network Service Registry Domains.


References
----------

* The documentation call [meeting minutes](https://docs.google.com/document/d/1113nzdL-DcDAWT3963IsS9LeekgXLTgGebxPO7ZnJaA/edit#)
* [NSM Glossary doc](https://docs.google.com/document/d/1zQOIQySDPgk2uUUKAWW34GiO0GVOwsN0qQA9tvSn97A/edit#)
