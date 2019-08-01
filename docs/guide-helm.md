# Using Helm to install NSM on kubernetes cluster

This document will show you how to use `Helm` for `NSM` installation. 

## Helm installation
[Helm Installation Guide](https://helm.sh/docs/using_helm/#quickstart-guide)

## Useful Helm commands
* `$ helm install PATH_TO_CHART` - install specified chart on cluster
* `$ helm ls` - list of deployed releases and their states
* `$ helm delete RELEASE_NAME` - delete release

## Using Helm for NSM installation

Installing NSM with helm in the `nsm-system` namespace is as easy as:

```bash
$ helm install --namespace=nsm-system deployments/helm/nsm
```

*Note: in case of `Error: no available release name found` do (according to [issue](https://github.com/helm/helm/issues/4412)):*
```bash
$ kubectl create serviceaccount --namespace kube-system tiller
$ kubectl create clusterrolebinding tiller-cluster-rule --clusterrole=cluster-admin --serviceaccount=kube-system:tiller
$ kubectl patch deploy --namespace kube-system tiller-deploy -p '{"spec":{"template":{"spec":{"serviceAccount":"tiller"}}}}'
```

## Using Helm to install examples
After installation of NSM on cluster you can install examples to check correctness of cluster configuration.

Install simple NSC and icmp-responder:
```
helm install deployments/helm/icmp-responder
```

Install vppagent-nsc and vppagent-icmp-responder:
```
helm install deployments/helm/vpp-icmp-responder
```

Install vpn-gateway-nsc, vpp-gateway-nse and vppagent-firewall-nse:
```
helm install deployments/helm/vpn
```

Install skydive, crossconnect-monitor and jaeger:
```
helm install --namespace=nsm-system deployments/helm/nsmd-monitoring
```

## Values specification
Every chart has file `values.yaml`. Now there is a possibility to specify docker registry and tag for images in this file:

```yaml
registry: docker.io
org: networkservicemesh
tag: master
```