Excluded prefixes service
============================

Specification
-------------

Excluded prefixes are needed to detect IP addresses clashes when an IP from a remote network, 
either client's or endpoint's can accidentally overlap with the local (node's) network address space.

Prefix service is designed to collect the local IP address ranges. Additionally, user defined
address ranges could be configured to be treated as local.  

Implementation details
---------------------------------

Prefix service is deployed as dedicated pod in cluster, it monitors cluster's network via subscribing for:
* Kubernetes node events: NodeInterface.Watch
* Kubernetes service events: ServiceInterface.Watch

See prefixcollector.monitorReservedSubnets() for details.

 When changes are detected in cluster's network configuration the prefix service updates 
 'excluded_prefixes.yaml' property of nsm-config ConfigMap. 

nsm-config ConfigMap is designed to store cluster-wide information to be used by 
all NSM managers on the cluster. To access information in nsm-config in a 
non Kubernetes-aware way, a pod shell mount the ConfigMap as a volume, after that all properties 
in the ConfigMap get projected as ordinary files in the mounted directory. 

See https://kubernetes.io/docs/tasks/configure-pod-container/configure-pod-configmap/#add-configmap-data-to-a-volume for details.

Also, any changes in ConfigMap properties are (almost) instantly propagated to the 
mounted directory, and can be detected via file monitoring.

The excluded prefixes stored in nsm-config.excluded_prefixes.yaml are then used by 
ExcludedPrefixesService to check NSM requests validity.


References
----------

* Issue(s) reference - #1735
* PR reference - #1809
