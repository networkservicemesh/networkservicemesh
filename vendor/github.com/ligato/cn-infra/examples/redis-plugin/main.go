package main

import (
	"github.com/ligato/cn-infra/core"
	"github.com/ligato/cn-infra/datasync"
	"github.com/ligato/cn-infra/datasync/kvdbsync"
	"github.com/ligato/cn-infra/datasync/resync"
	"github.com/ligato/cn-infra/db/keyval"
	"github.com/ligato/cn-infra/db/keyval/redis"
	"github.com/ligato/cn-infra/flavors/connectors"
	"github.com/ligato/cn-infra/flavors/local"
)

// Main allows running Example Plugin as a statically linked binary with Agent Core Plugins. Close channel and plugins
// required for the example are initialized. Agent is instantiated with generic plugin (Status check, and Log)
// and example plugin which demonstrates use of Redis flavor.
func main() {
	// Init close channel used to stop the example
	exampleFinished := make(chan struct{}, 1)

	// Start Agent with ExamplePlugin, RedisPlugin & FlavorLocal (reused cn-infra plugins).
	agent := local.NewAgent(local.WithPlugins(func(flavor *local.FlavorLocal) []*core.NamedPlugin {
		redisPlug := &redis.Plugin{}
		redisDataSync := &kvdbsync.Plugin{}
		resyncOrch := &resync.Plugin{}

		redisPlug.Deps.PluginInfraDeps = *flavor.InfraDeps("redis", local.WithConf())
		resyncOrch.Deps.PluginLogDeps = *flavor.LogDeps("redis-resync")
		connectors.InjectKVDBSync(redisDataSync, redisPlug, redisPlug.PluginName, flavor, resyncOrch)

		examplePlug := &ExamplePlugin{closeChannel: &exampleFinished}
		examplePlug.Deps.PluginLogDeps = *flavor.LogDeps("redis-example")
		examplePlug.Deps.DB = redisPlug          // Inject redis to example plugin.
		examplePlug.Deps.Watcher = redisDataSync // Inject datasync watcher to example plugin.

		return []*core.NamedPlugin{
			{redisPlug.PluginName, redisPlug},
			{redisDataSync.PluginName, redisDataSync},
			{resyncOrch.PluginName, resyncOrch},
			{examplePlug.PluginName, examplePlug}}
	}))
	core.EventLoopWithInterrupt(agent, exampleFinished)
}

// ExamplePlugin to depict the use of Redis flavor
type ExamplePlugin struct {
	Deps // plugin dependencies are injected

	closeChannel *chan struct{}
}

// Deps is a helper struct which is grouping all dependencies injected to the plugin
type Deps struct {
	local.PluginLogDeps                             // injected
	Watcher             datasync.KeyValProtoWatcher // injected
	DB                  keyval.KvProtoPlugin        // injected
}

// Init is meant for registering the watcher
func (plugin *ExamplePlugin) Init() (err error) {
	//TODO plugin.Watcher.Watch()

	return nil
}

// AfterInit is meant to use DB if needed
func (plugin *ExamplePlugin) AfterInit() (err error) {
	db := plugin.DB.NewBroker(keyval.Root)
	db.ListKeys(keyval.Root)

	return nil
}

// Close is called by Agent Core when the Agent is shutting down. It is supposed to clean up resources that were
// allocated by the plugin during its lifetime
func (plugin *ExamplePlugin) Close() error {
	*plugin.closeChannel <- struct{}{}
	return nil
}
