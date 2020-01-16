# NSM v0.2.0 Borealis - Release notes

## Version 0.2.0
Thanks go to our community and sponsors for their improvements, new features, feedback and guidance.

## New features and improvements:
1. **DNS support** \
Provides a workload with DNS service from Network Services without breaking K8s DNS. [DNS docs](https://github.com/networkservicemesh/networkservicemesh/blob/master/docs/spec/dns-integration.md).
2. **Interdomain NSM** \
Interdomain provides an ability for the client in one domain to consume a Network Service provided by an endpoint in another domain. [Interdomain docs](https://github.com/networkservicemesh/networkservicemesh/blob/master/docs/spec/interdomain.md).
3. **Kernel Forwarding Plane** \
Provides an alternative forwarding plane that leverages the built-in tools provided by the Kernel. [Kernel forwarding plane docs](https://github.com/networkservicemesh/networkservicemesh/blob/master/forwarder/kernel-forwarder/README.md).
4. **Security** \
Utilized GRPC for messaging. GRPC can be secured using TLS. Network Service Mesh should utilize TLS using certificates issued by some authority. [Security docs](https://github.com/networkservicemesh/networkservicemesh/blob/master/docs/spec/security.md).
5. **Expose Discovery Service to NSCs** \
There will be occasions when NSCs (particularly those which are also NSEs) may need to make more sophisticated and programmatic decisions than can be expressed in the NS resource about Endpoint Selection. This allows them to discover the NSEs to which to apply that criteria. [PR#1825](https://github.com/networkservicemesh/networkservicemesh/pull/1825)


## Backwards Compatibility:
1. Updated Kubernetes to v1.16.2
2. Deployment fully migrated to helm
3. Dataplane is renamed to Forwarder. You now need to use `vpp-forwarder` instead of `vpp-dataplane`. For details, please see [Forwarder docs](https://github.com/networkservicemesh/networkservicemesh/blob/master/forwarder/README.md).
4. API Changes: Unified {local,remote}.NetworkService API to simply NetworkService API
5. We recommend using Kind as a provisioning tool. Vagrant is still suported, but is recommended mainly for developer purposes.

## Bug fixes:
1. The vpp-agent was not sending metrics previously. Fixed by [vpp-agent/PR#1495](https://github.com/ligato/vpp-agent/pull/1495)

## Future changes:
1. We are going to upgrade to Helm v3.0.0.
