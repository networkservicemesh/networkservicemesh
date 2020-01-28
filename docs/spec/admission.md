NSM Mutating Admissions Controller
============================

Specification
-------------

When Kubernetes admits a new Resource, like Pods, there is a process of [Admission Controllers](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/) that can either deny their admission, or *mutate* them.

**Network Service Mesh** requires certain incantations in Pods taking advantage of NSM.  This presents a certain barrier to entry.  A [mutating admissions webhook](https://banzaicloud.com/blog/k8s-admission-webhooks/) can be used to look at new Pods prior to admission, and add the NSM incantations.

## What to trigger on?
A Pod can have in its metadata section `annotations`, which are key value pairs (similar to labels):

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: icmp-responder-nsc
  annotations:
    ns.networkservicemesh.io: icmp-responder
```

The annotation key, in this case, is `ns.networkservicemesh.io`, the value is `icmp-responder`.

The basic form of the value is a comma-delimited list of *`urls`* for the network service:

```sh
${nsname}/${optionally interface to attach the ns to}?${optional & delimited list of labels}
```

So for example:

```sh
secure-intranet-connectivity/eth2?app=firewall&version=2
```

Would imply a network service named `secure-intranet-connectivity` connected on `eth2`, with labels: `app=firewall` and `version=2`.

Merging this example to the full `yaml` above, we can let the client connect with two network services simultaneously:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: icmp-responder-nsc
  annotations:
    ns.networkservicemesh.io: icmp-responder,secure-intranet-connectivity/eth2?app=firewall&version=2
```

NOTE: The interface part cannot exceed 15 chars, and if it does a *really* clear error should result.

## The Results of the Mutation Admission Webhook

If and only if the Pod has the `ns.networkservicemesh.io` annotation exists, and is of the right form, then we should add to the Pod spec a patch with the following content:

```yaml
      initContainers:
      - name: nsm-init-container
        image: ${REPO}/${INITCONTAINER}:${TAG}
        imagePullPolicy: IfNotPresent
        env:
        - name: NS_NETWORKSERVICEMESH_IO
          value: ${value of annotation}
        resources:
          limits:
            networkservicemesh.io/socket: 1
```

The ${INITCONTAINER} is based on the SDK's `NSMClientList` which parses `${NS_NETWORKSERVICEMESH_IO}` and spawns the needed number of clients.

${REPO}, ${INITCONTAINER}, and ${TAG} are specifiable for the mutating admission webhook container, defaulting to REPO=networkservicemesh, INITCONTAINER=nsc, TAG=latest.

The nsm-init container will be added to the beginning of the `initContainers` list of the POD. It means that other init containers on the list can do some work with a created connection/network setup prepared by `nsm-init-container`.
NOTE: Depending on the value of annotation `ns.networkservicemesh.io` NSM init container can prepare multiple connections (see the merge example above).
## Possible Augmentations

Because the Mutating Admission Controller allows us to add complexity to the initcontainer without taxing the user, it is desirable to have the initcontainer add additional information, for example, the Pod id, or Node name via the downward API as env variables that can be then added as labels to the Network Service Request.  This will likely be handy for #708.


Implementation details
----------------------

None

Example usage
-------------

### Configuring the admission controller

Following is an example of the full NSM admission controller deployment.
Here ${CA_BUNDLE} is an environment variable to hold the pre-created CA bundle.

```yaml
---
apiVersion: admissionregistration.k8s.io/v1beta1
kind: MutatingWebhookConfiguration
metadata:
  name: nsm-admission-webhook-cfg
  labels:
    app: nsm-admission-webhook
  namespace: nsm-system
webhooks:
  - name: admission-webhook.networkservicemesh.io
    clientConfig:
      service:
        name: nsm-admission-webhook-svc
        namespace: nsm-system
        path: "/mutate"
      caBundle: ${CA_BUNDLE}
    rules:
      - operations: ["CREATE"]
        apiGroups: ["apps", "extensions", ""]
        apiVersions: ["v1", "v1beta1"]
        resources: ["deployments", "services", "pods"]
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nsm-admission-webhook
  labels:
    app: nsm-admission-webhook
  namespace: nsm-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nsm-admission-webhook
  template:
    metadata:
      labels:
        app: nsm-admission-webhook
    spec:
      containers:
        - name: nsm-admission-webhook
          image: networkservicemesh/admission-webhook
          imagePullPolicy: IfNotPresent
          volumeMounts:
            - name: webhook-certs
              mountPath: /etc/webhook/certs
              readOnly: true
          env:
          - name: REPO
              value: networkservicemesh
          - name: INITCONTAINER
              value: nsc
          - name: TAG
              value: latest
      volumes:
        - name: webhook-certs
          secret:
            secretName: nsm-admission-webhook-certs
---
apiVersion: v1
kind: Service
metadata:
  name: nsm-admission-webhook-svc
  labels:
    app: nsm-admission-webhook
  namespace: nsm-system
spec:
  ports:
    - port: 443
      targetPort: 443
  selector:
    app: nsm-admission-webhook
```

### Annotated client

Following is an example of an NSM annotated pod based on `alpine` container. If the admission webhook is installed it will inject an inticontainer (according to webhook's configuration) and will request a network service `icmp-responder` labelled with `app=icmp`.

```yaml
---
apiVersion: apps/v1
kind: Deployment
spec:
  selector:
    matchLabels:
      networkservicemesh.io/app: "alpine-pod"
  template:
    metadata:
      labels:
        networkservicemesh.io/app: "alpine-pod"
    spec:
      containers:
        - name: alpine-img
          image: alpine:latest
          command: ['tail', '-f', '/dev/null']
metadata:
  name: alpine-pod
  annotations:
    ns.networkservicemesh.io: icmp-responder?app=icmp
```

References
----------

* Issue(s) reference - #708
* PR reference - #723
