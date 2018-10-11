## CN-Infra Core

The `core` package contains the CN-Infra Core that manages the startup
and graceful shutdown of CN-Infra based applications. The startup & 
shutdown lifecycle is depicted in the sequence diagram below. The startup
and shutdown behavior is described in comments for the `Start()` and 
`Stop()` functions in [agent_core.go](agent_core.go), and for the 
`EventLoopWithInterrupt()`function in [event_loop.go](event_loop.go).

![plugin lifecycle](../docs/imgs/plugin_lifecycle.png)

The `core` package also defines the CN-Infra Core's [SPI](plugin_spi.go)
that must be implemented by each plugin (see [Guidelines](../docs/guidelines/PLUGIN_LIFECYCLE.md)). 
The SPI is used by the Core to Init(), AfterInit() and Close() each plugin. 
 




