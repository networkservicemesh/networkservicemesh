# Configuring Network Service Mesh with Environment Variables

## Any container, which supports opentracing
* *TRACER_ENABLED* - Represents boolean. Disables opentracing if false. False by default.

## NSMgr

**NSMD**

* *NSMD_API_ADDRESS* - Specifies IP address and port to start NSMD server (default ":5001")
* *INSECURE* - Allows to start NSMD in insecure mode (all `grpc.Dial()` will be called with `grpc.WithInsecure()`)

**NSMD-K8S**

* *PROXY_NSMD_K8S_ADDRESS* - Proxy NSMD-K8S service address to forward Network Service discovery request (default "pnsmgr-svc:5005")

## Proxy NSMgr

**PROXY NSMD**

* *PROXY_NSMD_API_ADDRESS* - Specifies IP address and port to start Proxy NSMD server (default ":5006")
* *PROXY_NSMD_K8S_ADDRESS* - Proxy NSMD-K8S service address and port (default "pnsmgr-svc:5005")
* *PROXY_NSMD_K8S_REMOTE_PORT* - Kubernetes node port, NSMD-K8S service forwarded to (default "80")

**PROXY NSMD-K8S**

* *PROXY_NSMD_ADDRESS* - Proxy NSMD service address and port (default "pnsmgr-svc:5006")
* *PROXY_NSMD_K8S_REMOTE_PORT* - Kubernetes node port, NSMD-K8S service forwarded to (default "80")

## NSM-MONITOR
* *MONITOR_DNS_CONFIGS* - Means boolean flag. If the flag is true then nsm-monitor will monitor DNS configs.

## NSM-COREDNS
* *USE_UPDATE_API* - Means boolean flag. If the flag is true then nsm-coredns will accept dns configs to dynamically change Corefile. With this flag, nsm-coredns can be deployed without corefile.
* *UPDATE_API_CLIENT_SOCKET* - Represents the path to the client socket.

##NSM-ADMISSION-WEBHOOK
* *DNS_SEARCH_DOMAINS* - Represents a list of strings. Uses for configuring DNS Search domains patch.
