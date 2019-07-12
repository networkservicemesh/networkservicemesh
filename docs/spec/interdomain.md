Interdomain NSM
============================

Specification
-------------

Interdomain provides an ability for the Client in one domain consume a Network Service provided by an Endpoint in another domain.

Implementation details
---------------------------------

1. First, the Client asking for a Network Service with a name of the form: "network-service@example.com"

2. Then the NSMgr sends the registry calls to find the NetworkService Endpoints for the Network Service to the proxy NSMgr in that domain.

3. Proxy NSMgr resolves domain name via DNS and sends a GRPC Request to the remote domain NSMgr to find NetworkService Endpoints for the Network Service

4. Client's NSMgr negotiate the 'remote.Mechanism' with remote domain NSMgr through the proxy NSMgr in Client's domain.

5. Both sides setup their end of the tunnel.
    *  Client's NSMgr initiate connection monitoring via proxy NSMgr for the death and healing purposes

Example usage
------------------------

Take a look an example in interdomain integration tests

References
----------

* Issue(s) reference - https://github.com/networkservicemesh/networkservicemesh/issues/714
* PR reference - https://github.com/networkservicemesh/networkservicemesh/pull/1298