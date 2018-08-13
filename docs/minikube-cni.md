Using CNI Plugins With Minikube
-------------------------------

Minikube claims support for CNI plugins via the CLI, and this information
can be found [here](https://kubernetes.io/docs/setup/minikube/). However,
note that as of August, 2018, there is a [bug](https://github.com/kubernetes/minikube/issues/2828)
which requires a workaround. The workaround is as follows:

Start Minikube as follows:

`minikube start --network-plugin=cni --extra-config=kubelet.network-plugin=cni`

At this point, install your CNI plugin into Minikube. In this example,
Calico was installed and can be shown working as follows:

```
kubectl get pods --all-namespaces
NAMESPACE     NAME                                       READY     STATUS    RESTARTS   AGE
kube-system   calico-etcd-wjpgt                          1/1       Running   0          1m
kube-system   calico-node-s1mw3                          2/2       Running   0          1m
kube-system   calico-policy-controller-336633499-j9j23   1/1       Running   0          1m
kube-system   kube-addon-manager-minikube                1/1       Running   0          2m
kube-system   kube-dns-910330662-4l368                   3/3       Running   0          2m
kube-system   kubernetes-dashboard-36hv5                 1/1       Running   0          2m
```

