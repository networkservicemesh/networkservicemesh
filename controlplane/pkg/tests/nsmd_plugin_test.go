package tests

import (
	"context"
	"os"
	"path"
	"testing"

	pluginsapi "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/plugins"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/plugins"
)

func TestPluginReRegistration(t *testing.T) {
	const pluginName = "dummy-plugin"

	registryDir := path.Join(os.TempDir(), "networkservicemesh")
	if err := os.MkdirAll(registryDir, os.ModeDir|os.ModePerm); err != nil {
		t.Fatalf("Failed to create socket folder for Plugin Registry: %v", err)
		return
	}
	defer func() { _ = os.RemoveAll(registryDir) }()

	registryPath := path.Join(registryDir, path.Base(pluginsapi.PluginRegistrySocket))

	pluginRegistry := plugins.NewPluginRegistry(registryPath)
	if err := pluginRegistry.Start(context.Background()); err != nil {
		t.Fatalf("Failed to start Plugin Registry: %v", err)
		return
	}

	defer func() {
		if err := pluginRegistry.Stop(); err != nil {
			t.Errorf("Failed to stop Plugin Registry: %v", err)
		}
	}()

	plugin := &dummyConnectionPlugin{
		prefixes: []string{"10.10.1.0/24"},
	}
	services := make(map[pluginsapi.PluginCapability]interface{}, 1)
	services[pluginsapi.PluginCapability_CONNECTION] = plugin

	if err := plugins.StartPlugin(pluginName, registryPath, services); err != nil {
		t.Fatalf("Failed to start first instance of dummy plugin: %v", err)
	}

	if err := plugins.StartPlugin(pluginName, registryPath, services); err != nil {
		t.Fatalf("Failed to start second instance of dummy plugin: %v", err)
	}
}
