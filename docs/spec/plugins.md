Plugins
=======

Specification
-------------

NSM is a cluster agnostic technology and any cluster specific functionality could be provided via extensible plugins mechanism. Here is a small glossary:

- **Plugin** — a gRPC server that has one or more Plugin Capabilities to extend NSM functionality
- **Plugin Capability** — a gRPC service definition that specifies methods related to one small piece of NSM functionality (for example, the connection capability is only responsible for updating and validating a connection)
- **Plugin Registry** — NSM service that registers plugins and provides a way to call them from NSM

NSM Plugin Registry has a registration gRPC service, so each plugin can register itself through this service. The communication between NSM Plugin Registry and plugins is happen through gRPC on a Unix socket.

Each plugin is a gRPC server that implements one (or more) of the defined gRPC services (plugin capabilities). The plugin has to start the server on a Unix socket under `plugins.PluginRegistryPath` and send the socket path on registration. Once the plugin is registered, it can serve incoming gRPC requests from NSM.

Implementation details
----------------------

The model is placed in `controlplane/api/plugins` directory. It contains the following files:
- **constants.go** specifies `PluginRegistryPath` (the location of plugin sockets) and `PluginRegistrySocket` (the location of NSM Plugin Registry socket) constants
- **registry.proto** defines Plugin Registry gRPC service
- **connectionplugin.proto** defines a gRPC model for plugins have the connection capability

Plugin Registry implementation is placed in `controlplane/pkg/plugins` directory. It contains the following files:
- **registry.go** implements Plugin Registry that can register plugins and provide getters for plugin managers
- **connectionplugin.go** implements a connection plugin manager that can call all plugins have the connection capability

Plugin Registry is stored as a field inside **nsm.NetworkServiceManager** implementation and may be called in the following way:

```go
import "github.com/networkservicemesh/networkservicemesh/controlplane/api/plugins"

...

func (srv *networkServiceManager) updateConnection(ctx context.Context, conn connection.Connection) (connection.Connection, error) {
    ...
    
    info, err := srv.pluginRegistry.GetConnectionPluginManager().UpdateConnection(ctx, plugins.NewConnectionWrapper(conn))
    if err != nil {
        return nil, err
    }
    
    return info.GetConnection(), nil
}

func (srv *networkServiceManager) validateConnection(ctx context.Context, conn connection.Connection) error {
    ...
    
    result, err := srv.pluginRegistry.GetConnectionPluginManager().ValidateConnection(ctx, plugins.NewConnectionWrapper(conn))
    if err != nil {
        return err
    }
    
    if result.GetStatus() != plugins.ConnectionValidationStatus_SUCCESS {
        return fmt.Errorf(result.GetErrorMessage())
    }
    
    return nil
}
```

Example usage
-------------

#### 1. Create a gRPC server implements a plugin

Create a gRPC server on a Unix socket under `plugins.PluginRegistryPath` path. Then register it with a service by calling `plugins.Register*PluginServer(server, service)`.

If you implement a plugin with more than one capability, you have to register it few times. Check `plugins.PluginCapability` enum to see the list of supported capabilities.

#### 2. Register the plugin in NSM Plugin Registry

NSM Plugin Registry is a gRPC server run on the Unix socket at `plugins.PluginRegistrySocket` path. To register a plugin you have to make a gRPC call to the `Register` method and provide registration data.

Registration data is provided in `plugins.PluginInfo` structure which has the following fields:
- **Name** - your plugin name
- **Endpoint** — the path to the Unix socket you've started gRPC server on
- **Capabilities** — list of capabilities supported by your plugin

```go
import "github.com/networkservicemesh/networkservicemesh/controlplane/api/plugins"

...

// 1. Create a gRPC server that implements a plugin

endpoint := path.Join(plugins.PluginRegistryPath, "my-plugin.sock")
sock, err := net.Listen("unix", endpoint)
if err != nil {
    return err
}

server := grpc.NewServer()

service := newConnectionPluginService()

plugins.RegisterConnectionPluginServer(server, service)

go func() {
    if err := server.Serve(sock); err != nil {
        logrus.Error("Failed to start Plugin gRPC server", endpoint, err)
    }
}()

// 2. Register the plugin in NSM Plugin Registry

conn, err := grpc.Dial("unix:"+plugins.PluginRegistrySocket)
defer conn.Close()
if err != nil {
    logrus.Fatalf("Cannot connect to the Plugin Registry: %v", err)
}

client := plugins.NewPluginRegistryClient(conn)

ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
defer cancel()

_, err = client.Register(ctx, &plugins.PluginInfo{
    Name:         "my-plugin",
    Endpoint:     endpoint,
    Capabilities: []plugins.PluginCapability{plugins.PluginCapability_CONNECTION},
})
if err != nil {
    return err
}
```

References
----------

* Issue(s) reference - [#1339](https://github.com/networkservicemesh/networkservicemesh/issues/1339)
* PR reference - [#1356](https://github.com/networkservicemesh/networkservicemesh/pull/1356)
