
**<span style="text-decoration:underline;">Introduction</span>**

These are the instructions to deploy an instance of Network Service Mesh into GKE.

**<span style="text-decoration:underline;">Requirements</span>**



1.  A GKE Account
1.  A GKE project, for the examples below the project name will be: "**nsm-gke-project**"

**<span style="text-decoration:underline;">Steps </span>**

Build a docker image from the source code

   	


```SHELL
 $ sudo docker build -t gcr.io/nsm-gke-project/netmesh:0.0.1 -f build/nsm/docker/Dockerfile .
```


Publish the image to the google cloud repository


```
 $ gcloud docker -- push gcr.io/nsm-gke-project/netmesh:0.0.1
```


Edit the file conf/sample/networkservice-daemonset.yaml to change the image repository location.


```
containers:
        - name: netmesh
          image: gcr.io/nsm-gke-project/netmesh:0.0.1
          imagePullPolicy: IfNotPresent
```



```YAML
apiVersion: extensions/v1beta1
kind: DaemonSet
metadata:
  name: networkservice
spec:
  template:
    metadata:
      labels:
        app: networkservice-ds
    spec:
      nodeSelector:
        beta.kubernetes.io/arch: amd64
      serviceAccountName: networkservicemesh
      containers:
        - name: netmesh
          image: gcr.io/gcp-eng-dev/netmesh:0.0.1
          imagePullPolicy: IfNotPresent

---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: networkservicemesh

---
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: crd-creater
rules:
- apiGroups: ["*"]
  resources: ["*"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]

---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: crd-creater-binding
subjects:
- kind: ServiceAccount
  namespace: default
  name: networkservicemesh
roleRef:
  kind: ClusterRole
  name: crd-creater
  apiGroup: rbac.authorization.k8s.io
```


Set the IAM roles for the project to have "Kubernetes Engine Admin" role

Create a Kubernetes cluster in GKE


```
$ gcloud beta container --project "nsm-gke-project" clusters create "nsm-test" --zone "us-west1-a" --username "admin" --cluster-version "1.9.6-gke.1" --machine-type "n1-standard-1" --image-type "UBUNTU" --disk-type "pd-standard" --disk-size "100" --service-account "nsm-gke-project@nsm-gke-project.iam.gserviceaccount.com" --enable-kubernetes-alpha --num-nodes "1" --enable-cloud-logging --enable-cloud-monitoring --network "default" --subnetwork "default" --addons HorizontalPodAutoscaling,HttpLoadBalancing,KubernetesDashboard
```


Get the gcloud login and set it in your terminal


```
$ kubectl create clusterrolebinding "username"-cluster-admin-binding --clusterrole=cluster-admin --user="user email address" 
```


# Create the daemonset 


```
$ kubectl create -f conf/sample/networkservice-daemonset.yaml
```


# Create the channel


```
$ kubectl create -f conf/sample/networkservice-channel.yaml
```


# Create the endpoints


```
$ kubectl create -f conf/sample/networkservice-endpoint.yaml
```


# Finally, create the network service


```
$ kubectl create -f conf/sample/networkservice.yaml
```



```
$ kubectl get pods, crd, NetworkService, NetworkServiceChannel, NetworkServiceEndpoint

```

```
NAME                      READY     STATUS    RESTARTS   AGE
po/networkservice-jbwdw   1/1       Running   0          2h

NAME                                                                      AGE
customresourcedefinitions/networkservicechannels.networkservicemesh.io    2h
customresourcedefinitions/networkserviceendpoints.networkservicemesh.io   2h
customresourcedefinitions/networkservices.networkservicemesh.io           2h

NAME                           AGE
networkservices/gold-network   10s

NAME                                   AGE
networkservicechannels/gold-ethernet   53s

NAME                                      AGE
networkserviceendpoints/gold-endpoint-1   24s
```

