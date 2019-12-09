
Endpoint selection for clients 
============================

Specification
-------------

When a client requests a connection to Network Service mesh they are matched to an endpoint based on label selectors. Typically these label selectors are static eg: app=firewall.

Dynamic label selection with templating allow clients to be connected to endpoints based on underlying infrastructure, e.g: connect to an endpoint running on the same underlying node. Dynamic label selection is achieved via templating in the DestinationSelector only currently.

NodeName will select matching endpoints on the same Node as the client making the request

``` "nodeName": "{{index . \"nodeName\"}}",```



Example usage
------------------------

Take a look at tests in **match_selector_test.go** 

References
----------

* Issue(s) reference - https://github.com/networkservicemesh/networkservicemesh/issues/1824
