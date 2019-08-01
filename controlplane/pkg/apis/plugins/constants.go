package plugins

const (
	// PluginRegistryPath defines the location of NSM Plugin sockets
	PluginRegistryPath = "/var/lib/networkservicemesh/plugins/"
	// PluginRegistrySocket defines the location of NSM Plugin Registry socket
	PluginRegistrySocket = PluginRegistryPath + "registry.sock"
)
