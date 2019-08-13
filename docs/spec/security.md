Security
============================

##Specification
Network Service Mesh utilizes GRPC for messaging. GRPC can be secured using TLS. Network Service Mesh 
should utilize TLS using certificates issued by some authority. 

But TLS is merely used for securing the connection, not for fundamental authentication or authorization due 
to its inability to properly handle message provenance. JWT tokens will be used for such purposes.

#### Identity

Network Service Mesh identities are expressed with SPIFFE SVIDs.  
Fundamentally, SPIFFE expresses [identity](https://github.com/spiffe/spiffe/blob/master/standards/SPIFFE-ID.md) as:

`spiffe://trust-domain/path`

In a Kubernetes environment, the trust domain might be the cluster, 
and the path portion indicating a particular workload within the cluster.

[Spire](https://spiffe.io/spire/) provides a reference implementation for SPIFFE.  When deploying to Kubernetes Spire can map Kubernetes service accounts (sa) to identities. 
It is anticipated for the NSM Kubernetes Implementation we will take advantage of this approach.

#### Transport Security

Every internal `grpc.Dial()` and `grpc.NewServer()` should be secured using `TransportCredentials`. 

#### Provenance
*coming soon...*

## Implementation details
Spire consist of two components: 
* ***spire-agent*** - DaemonSet, has instances on every node, responsible for workload attestation, provides unix socket for certificate obtaining
* ***spire-server*** - StatefulSet, one per cluster, responsible for *spire-agent* attestation, store information about SPIFFE ID

In order to obtain certificates workload has to mount unix socket
`/run/spire/sockets/agent.sock`. All volumes will be mounted by AdmissionWebhook in case 
it discovers annonation `security.networkservicemesh.io: ""`

#### How to add workload and new SPIFFE ID

1. Choose ServiceAccount for workload, create new or use existing one.
2. Add new entry to `security/conf/registration.json`:
    ```json
    {
      "entries": [
        {
          "selectors": [
            {
              "type": "k8s",
              "value": "sa:workload-service-account"
            }
          ],
          "spiffe_id": "spiffe://test.com/workload",
          "parent_id": "spiffe://test.com/spire-agent"
        }
      ]
    }
    ```
3. Rebuild spire-registration image:
    ```bash
    $ make docker-spire-registration-build
    ```
4. Modify workload's yaml, add *serviceAccount* and security annotation:
    ```yaml
    apiVersion: apps/v1
    kind: Deployment
    spec:
      template:
        spec:
          serviceAccount: workload-service-account
          containers:
            - name: alpine-img
              image: alpine:latest
              command: ['tail', '-f', '/dev/null']
    metadata:
      name: workload-alpine
      annotations:
        security.networkservicemesh.io: ""
    
    ```