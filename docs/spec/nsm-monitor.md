
Sidecar nsm-monitor
============================

Specification
-------------
`nsm-monitor` is simple sidecar for monitoring connection events into client's POD. Also, it providing API for creating custom `nsm-monitor` (see below). 
Monitoring for connection can be used for different purposes, for example:
* Extract data from the connection
* Handle events related to connection removed/added/healed.

Implementation details
---------------------------------

#### How to start to monitor into client's POD
For monitoring client's connection events you need to add into POD additional container. For example:
```
...
spec:
    spec:
      hostPID: true
      containers:
        #original container
        - name: vppagent-nsc
          image: networkservicemesh/vpp-test-common:latest
          imagePullPolicy: IfNotPresent
        #injected monitor
        - name: nsm-monitor
          image: networkservicemesh/nsm-monitor:lateest
          imagePullPolicy: IfNotPresent
...

```

#### How to create custom nsm-monitor
For creating custom nsm-monitor you need to implement next interface: 
```
// NSMMonitorHandler - handler to perform configuration of monitoring app
type NSMMonitorHandler interface {
	//Connected occurs when the nsm-monitor connected
	Connected(map[string]*networkservice.Connection)
	//Closed occurs when the connection closed
	Closed(conn *networkservice.Connection)
	//Healing occurs when the healing started
	Healing(conn *networkservice.Connection)
	//GetConfiguration gets custom network service configuration
	GetConfiguration() *common.NSConfiguration
	//ProcessHealing occurs when the restore failed, the error pass as the second parameter
	ProcessHealing(newConn *networkservice.Connection, e error)
	//Stopped occurs when the invoked NSMMonitorApp.Stop()
	Stopped()
	//IsEnableJaeger returns is Jaeger needed
	IsEnableJaeger() bool
}

```
After that, you could use next code for your sidecar:
```
func main() {
    c := tools.NewOSSignalChannel()
    app := nsm_sidecars.NewNSMMonitorApp()
    app.SetHelper(newMyHepler())
    go app.Run(version)
    <-c
}
```

Example usage
------------------------
For an example of usage you could take a look at tests:

* TestNSMMonitorInit
* TestDeployNSMMonitor
* test/applications/cmd/monitoring-nsc
* test/applications/cmd/monitoring-dns-nsc

References
----------

* [Spec: ResiliencyV2](https://github.com/networkservicemesh/networkservicemesh/issues/1331) scenario №9.
* [Soec: Spec: DNS Integration for NSM](https://github.com/networkservicemesh/networkservicemesh/issues/1224) scenario №1
