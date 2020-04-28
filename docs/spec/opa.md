Open Policy Agent
============================

Specification
-------------

The [Open Policy Agent](https://www.openpolicyagent.org/docs/latest/) decouples policy decision-making from policy enforcement. When your software needs to make policy decisions it **queries** OPA and supplies structured data (e.g., JSON) as input. OPA accepts arbitrary structured data as input. 

Network Service Mesh utilizes OPA for deploying something policies cases. For using OPA policies we provide **[authorization chain element](https://github.com/networkservicemesh/sdk/tree/master/pkg/networkservice/common/authorize)**.

Implementation details
---------------------------------
### OPA input object
OPA generates policy decisions by evaluating the query input and against policies and data.

In our implementation OPA input provides the following objects:
1. The Connection (usage input.connection.[...])
2. The TLSInfo (as retrieved by peer.FromContext for Server or grpc.Peer for the Client) which contains the following data: 
	*	"certificate"  --  is  a  pem  encoded  x509cert  (usage:  input.auth_info.certificate)
	*	"spiffe_id"  --  is  a  spiffeID  from  SVIDX509Certificate  (usage:  input.auth_info.spiffe_id)
3. "operation"  --  one  of  request/close  (usage:  input.operation)
4. "role"  --  one  of  client/endpoint  (usage:  input.role)

### OPA deployment mechanism

TODO

Example usage
------------------------

#### An example of using OPA input for the case of token signature verification

```
package test   
default allow = false    
allow { 
	token := input.connection.path.path_segments[0].token  
	cert := input.auth_info.certificate  
    io.jwt.verify_es256(token, cert) # signature verification  
}
```

#### Example of usage the deployment mechanism

TODO

#### Default OPA Policies 

TODO

References
----------

* Issue(s) reference - #200
* PR reference - #225
