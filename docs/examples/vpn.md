Network Service Mesh is capable of composing together many Endpoints to work together to provide the desired Network Service.
In the vpn example, the user wants **secure-intranet-connectivity** with the traffic from the App Pod Client passing through
first a firewall, and then two other passthrough security appliances before finally getting to a VPN Gateway.

## Deploy

Utilize the [Run](../guide-quickstart.md) instructions to install the NSM infrastructure, and then type:

```bash
helm install nsm/vpn
```

## What it does

This will install Deployments for:

Name | Advertises Network Service | Labels | Description |
:--------|:--------|:------ |:------
**vpn-gateway-nsc** | | | The Client
**vppagent-firewall-nse** | secure-intranet-connectivity | app=firewall |A passthrough firewall Endpoint
**vppagent-passthrough-nse-1** | secure-intranet-connectivity | app=passthrough-1 |A generic passthrough Endpoint
**vppagent-passthrough-nse-2** | secure-intranet-connectivity | app=passthrough-2 |A generic passthrough Endpoint
**vpn-gateway-nse** | secure-intranet-connectivity | app=vpn-gateway |A simulated VPN Gateway 

![vpn-example](../images/vpn-example.svg)

And also a Network Service:
```yaml
apiVersion: networkservicemesh.io/v1alpha1
kind: NetworkService
metadata:
  name: secure-intranet-connectivity
spec:
  payload: IP
  matches:
    - match:
      sourceSelector:
        app: "firewall"
      route:
        - destination:
          destinationSelector:
            app: "passthrough-1"
    - match:
      sourceSelector:
        app: "passthrough-1"
      route:
        - destination:
          destinationSelector:
            app: "passthrough-2"
    - match:
      sourceSelector:
        app: "passthrough-2"
      route:
        - destination:
          destinationSelector:
            app: "vpn-gateway"
    - match:
      route:
        - destination:
          destinationSelector:
            app: "firewall"

```

That describes how to compose together the various providers of Network Service **secure-intranet-connectivity**.

When the Client requests Network Service 'secure-intranet-connectivity with no labels:
![vpn-example-2](../images/vpn-example-2.svg)

it falls all the way through the **secure-intranet-connectivity** matches to:

```yaml
    - match:
      route:
        - destination:
          destinationSelector:
            app: "firewall"
```

And is connected to the Firewall Endpoint:

![vpn-example-3](../images/vpn-example-3.svg)

The Firewall Endpoint then requests **secure-intranet-connectivity** with labels **app=firewall**

![vpn-example-4](../images/vpn-example-4.svg)

and matches to:

```yaml
    - match:
      sourceSelector:
        app: firewall
      route:
        - destination:
          destinationSelector:
            app: "passthrough-1"
```

And gets wired to the Passthrough-1 Endpoint:

![vpn-example-5](../images/vpn-example-5.svg)

Which requests **secure-intranet-connectivity** with labels **app=passthrough-1**:

![vpn-example-6](../images/vpn-example-6.svg)

and matches to:
```yaml
    - match:
      sourceSelector:
        app: "passthrough-1"
      route:
        - destination:
          destinationSelector:
            app: "passthrough-2"
```

![vpn-example-7](../images/vpn-example-7.svg)

Which requests **secure-intranet-connectivity** with labels **app=passthrough-2**:

![vpn-example-8](../images/vpn-example-8.svg)

and matches to:

```yaml
    - match:
      sourceSelector:
        app: "passthrough-2"
      route:
        - destination:
          destinationSelector:
            app: "vpn-gateway"
```

![vpn-example-9](../images/vpn-example-9.svg)

## Verify

First verify that the vpn example Pods are all up and running:

```bash
kubectl get pods
```

To see the vpn example in action, you can run:

```bash
curl -s https://raw.githubusercontent.com/networkservicemesh/networkservicemesh/release-0.2/scripts/verify_vpn_gateway.sh | bash
```
