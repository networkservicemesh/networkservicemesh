# Plugin Flavors

## Flavor

The term **flavor** denotes a combination of plugins to be initialized
and used collectively in a CN-Infra based application.
It is implemented as a structure with plugins as members, i.e.:

```go
type Flavor struct {
       PluginA pluginA.Plugin
       PluginB pluginB.Plugin
       PluginC pluginC.Plugin

       injected bool // used to make sure the injection is performed only once (injection is explained below)
}
```

Additionally, flavor must implement method Plugins() returning plugins
in a list, as prescribed by the [Flavor interface](../../core/list_flavor_plugin.go).
This method is supposed to first resolve inter-plugin dependencies
through the approach of [dependency injection][1] and then call a helper
method [ListPluginsInFlavor](../../core/list_flavor_plugin.go), which
uses reflection to traverse top level fields of the flavor structure,
extracts all plugins and returns them as a slice of
[NamedPlugins](../../core/name.go).
While not being enforced by the interface, as an unwritten rule,
the code that handles the dependency injections is put separately into
a member function called **Inject()** that has no input parameters and
returns *true* if the injection was successful, *false* otherwise.
The Inject() method should also ensure that dependency injection
is performed at most once. For that purpose we add **injected** boolean
member to guard this condition.
Please follow the pattern of dependency injection and avoid any extra
magic with reflection for inter-connection of plugins.
For complex flavors, the resulting **Inject()** method may end up being
quite "chatty", but the added value is a better readability and an extra
level of verification from the compiler.

Skeleton of the flavor methods:

```go
func (f *Flavor) Inject() bool {
       if f.injected { // inject at most once
             return false
       }
       f.injected = true

       // perform the dependency injection here
       return true
}

func (f *Flavor) Plugins() []*core.NamedPlugin {
	flavor.Inject()
	return core.ListPluginsInFlavor(flavor)
}
```

## Reusable flavors

Some of the plugins are related and often used together. We have
therefore grouped them together and created flavors that can be inherited
from through the golang embedding. The most notable one is
the [Local flavor](../../flavors/local/local_flavor.go).
It includes [servicelabel](../../servicelabel/plugin_api_servicelabel.go),
[statuscheck](../../health/statuscheck/plugin_api_statuscheck.go) and
[Log registry](../../logging/log_api.go) and can be easily included into
another flavor as follows:

```go
type Flavor struct {
       *local.FlavorLocal
       //…
}
```

The members of included flavors will get loaded as if they were listed
directly, but this is much more concise.
Through the embedding of multiple flavors it may happen that some common
plugin is mentioned multiple types.
The [ListPluginsInFlavor()](../../core/list_flavor_plugin.go) method
used to make the “flattening”, i.e. converting flavor into the list
of plugins, makes sure that each plugin is loaded only once.
For this to work properly, however, reusable flavors have to be embedded
with pointers, otherwise the fields would get initialized by the compiler
before ListPluginsInFlavor() can execute duplicity-free flattening.
On the other hand, top-level plugins are embedded into flavors directly,
without pointers, as explained in
the [Plugin Dependencies Guidelines](PLUGIN_DEPENDENCIES.md).

## Guidelines

For your applications, you can either define your own flavor, like we do
in the [datasync-plugin example](../../examples/datasync-plugin/main.go),
or simply reuse existing flavor through embedding and extend it with your
own plugins. The latter approach can be seen in
the [logs-plugin example](../../examples/logs-plugin).
If your plugins do not require dependency injection, you can even
directly add them to the list of plugins as passed to NewAgent().


## Example

The example below demonstrates how a new flavor can be defined through
the composition of an existing flavor (RPC in this case) with
a user-defined plugin:

```go
package flavorexample

import (
	"github.com/ligato/cn-infra/core"
	"github.com/ligato/cn-infra/flavors/rpc"
	"github.com/ligato/cn-infra/rpc/rest"
)

type CompositeFlavor struct {
	*rpc.FlavorRPC    // Reused Flavor, notice the pointer!
	PluginXY PluginXY // Added custom plugin to flavor
}

func (flavor *CompositeFlavor) Inject() bool {
	if flavor.FlavorRPC == nil {
	    flavor.FlavorRPC = &rpc.FlavorRPC{}
	}
	if !flavor.FlavorRPC.Inject() {
	    return false
	}

	// Inject HTTP plugin from HTTP flavor into the user-defined plugin.
	// It requires some effort from the developer to write down all the injections one-by-one,
	// but the reward is a clear, human-readable, compiler-checked overview of all dependencies between the plugins from the flavor.
	flavor.PluginXY.HTTP = &flavor.FlavorRPC.HTTP

	// inject all other dependencies...

	return true
}

func (flavor *CompositeFlavor) Plugins() []*core.NamedPlugin {
	flavor.Inject()
	return core.ListPluginsInFlavor(flavor)
}


type PluginXY struct {
	Dep // plugin dependencies
}

type Dep struct {
	HTTP rest.HTTPHandlers // injected, this plugin just depends on the API interface
}

func (plugin* PluginXY) Init() error {
	// use injected dependency
	plugin.HTTP.RegisterHTTPHandler(...)

	return nil
}

func (plugin* PluginXY) Close() error {
	// do something
	return nil
}
```

[1]: https://en.wikipedia.org/wiki/Dependency_injection
