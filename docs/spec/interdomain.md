Interdomain NSM
============================

Specification
-------------

Interdomain provides an ability for the Client in one domain to consume a Network Service provided by an Endpoint in another domain.

Implementation details
---------------------------------

1. First, the Client asking for a Network Service with a name of the form: "network-service@example.com"

2. Then the NSMgr sends the registry calls to find the NetworkService Endpoints for the Network Service to the proxy NSMgr in that domain.

3. Proxy NSMgr resolves domain name via DNS and sends a GRPC Request to the remote domain NSMgr to find NetworkService Endpoints for the Network Service

4. Client's NSMgr negotiate the 'remote.Mechanism' with remote domain NSMgr through the proxy NSMgr in Client's domain.

5. Both sides setup their end of the tunnel.
    *  Client's NSMgr initiate connection monitoring via proxy NSMgr for the death and healing purposes

Client and endpoint nodes has to have public ipv4 addresses and have to be reachable by each other.

All remote mechanisms supported by NSM are also suported by Interdomain NSM.

Interdomain NSM does not have central registry. All clusters are communicate just within each single connection.

Network service can be reached by ipv4 format address and domain name. Currently domain name will be resolved by local DNS resolver and can be changed to any custom resolver ([func ResolveDomain(remoteDomain string)](../../k8s/pkg/utils/interdomainutils.go)).

Floating Interdomain
------------------------

Floating interdomain provides an ability to register Network Service Endpoints from any domain at one place, available from any another domain.

* NSMRS (Network Service Mesh Registry Server) is used as interdomain NSE registry server. 
* Floating Interdomain handle Network Services requests the same way as regular Interdomain request (for example request for Network Service of the form *network-service@nsmrs-domain.com*)
* NSMRS is independent from kubernetes (except spire registration).
* Proxy NSMD-K8S should be configured to forward registry packets to the NSMRS (Environment variable "*NSMRS_ADDRESS*").

Example usage
------------------------

Take a look an example in interdomain integration tests

Interdomain NSM supports and have been checked on Packet, AWS, AZURE and GKE clusters. 

References
----------

* Issue(s) reference - https://github.com/networkservicemesh/networkservicemesh/issues/714
* PR reference - https://github.com/networkservicemesh/networkservicemesh/pull/1298