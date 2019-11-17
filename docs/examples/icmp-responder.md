The simplest possible case for Network Service Mesh is to have is connecting a Client via a vWire to another Pod that is providing a Network Service.
We call this case the 'icmp-responder' example, because it allows the client to ping the IP address of the Endpoint over the vWire.


![icmp-responder-example](../images/icmp-responder-example.svg)

## Deploy

Utilize the [Run](/docs/setup/run/) instructions to install the NSM infrastructure, and then type:

```bash
helm install nsm/icmp-responder
```

## What it Does

This will install two Deployments:

Name | Description |
:--------|:--------
**icmp-responder-nsc** | The Clients, four replicas |
**icmp-responder-nse** | The Endpoints, two replicas |

And cause each Client to get a vWire connecting it to one of the Endpoints.  Network Service Mesh handles the
Network Service Discovery and Routing, as well as the vWire 'Connection Handling' for setting all of this up.

![icmp-responder-example-2](../images/icmp-responder-example-2.svg)

In order to make this case more interesting, Endpoint1 and Endpoint2 are deployed on two separate Nodes using
PodAntiAffinity, so that the Network Service Mesh has to demonstrate the ability to string vWires between Clients and
Endpoints on the same Node and Clients and Endpoints on different Nodes.

## Verifying

First verify that the icmp-responder example Pods are all up and running:

```bash
kubectl get pods | grep icmp-responder
```

To see the icmp-responder example in action, you can run:

```bash
curl -s https://raw.githubusercontent.com/networkservicemesh/networkservicemesh/release-0.2/scripts/nsc_ping_all.sh | bash
```
