# Release v1.4.1 (2018-07-23)

## Bugfix
  * Fixed issue in Consul client that caused brokers to incorrectly
  trim prefixes and thus storing invalid revisions for resync.

# Release v1.4 (2018-07-16)

## Breaking Changes
  * Package **etcdv3** was renamed to **etcd**. This change affects imports. Also 
  pay attention to the configuration flag which can also be influenced by the change.
  Based on the change, flag for ETCD configuration file `--etcdv3-config` is now defined 
  as `--etcd-config`.

## New Features
  * Support for GRPC unix domain socket type. Socket types tcp, tcp4, tcp6, unix 
  and unixpacket can be used with GRPC. Desired socket type and address/file can be 
  specified via grpc configuration file ([example here](rpc/grpc/grpc.conf)). More
  information in the [readme](rpc/grpc/README.md)
  * Rest plugin security improved. Security features are the usage of client/server certificates
  (HTTPS) and basic HTTP authentication with username and password. Those features 
  are disabled by default. Information about how to use it and example can be found in the 
  [readme](rpc/rest/README.md)
  
## Other
  * Example configuration files with description were added to every plugin which 
  supports/uses them. 
  
## Bugfix
  * Fixed occasional failure of method deriving config files
  * Fixed multiple issues in logs-lib example (logger, HTTP usage)  
  * To prevent incorrect values in subsequent changes, previous value of key should 
  be correctly cleaned up if the resync was called outside of initialization phase.
  * Fixed the logger configuration file. All created loggers are correctly set with a log
  level according to the map in file. The default log level can be also set, but keep in mind
  that the environmental variable `INITIAL_LOGLVL` replaces the value from the config.

# Release v1.3 (2018-05-24)

## New Features
  * Automatic resync if ETCD was disconnected and connected again. The feature is
  disabled by default. See [readme](db/keyval/etcd/README.md) to learn how to 
  enable the feature.
  * Watch registration [API](datasync/datasync_api.go) now contains new method 
  __Registration()__ allowing to register new key to all adapters.
  * New plugin for [Consul](db/keyval/consul). See [readme](db/keyval/consul/README.md)
  for more information.
  * In-memory mapping method __UpdateMetadata()__ now triggers events. Use __IsUpdate()__
  so see if the event comes from the update notification.
  
## Bugfix
  * Transport for statuscheck plugin fixed
  * Fixed bug where watcher was closed after server restart if database was compacted  

# Release v1.2 (2018-03-22)

## New Features
  * Added support for ETCD compacting. Information about how to use 
    it can be found in the [readme](db/keyval/etcd/README.md)
  * Name-to-index mapping API was extended with new method 'Update'.
    The purpose of the method is to update metadata value under specific
    key without triggering events, so mapping entry can be kept up to date. 

## Bugfix
  * Fixed syncbase issue where delete request for a non-existing item
    used to trigger a change notification
  * Getting of previous value for ProtoWatchResp 'delete' event now
    returns correct data  

# Release v1.1 (2018-02-07)

## Dependencies
  * Migrated from glide to dep

## Prometheus
  * Introduced Prometheus plugin with examples

# Release v1.0.8 (2018-01-22)

## Kafka
  * Added support for Kafka TLS.

## Logging
  * Logger config file now enables to set every logger to desired level or use default level for all loggers
    within plugins. For this purpose it is also possible to use environment variable INITITAL_LOGLVL.

## Statuscheck
  * Readiness probe now allows to report interfaces' state to the proble output. 

## Dependencies
  * Sirupsen package is now lower-cased according to recommandations.   

# Release v1.0.7 (2017-11-14)

## Agent, Flavors
Input arguments of `core.NewAgent()` were changed:
  * it is possible to call NewAgent without options: `core.NewAgent(flavor)`
  * you can pass options like this: `core.NewAgent(flavor, core.WithTimeout(1* time.Second))`
  * there is `core.NewAgentDeprecated()` for backward compatibility

This release contains utilities/options to avoid writing the new flavor go structures 
(Inject, Plugins methods) for simple customizations:
* if you just expose the RPCs you can write
  ```
  rpc.NewAgent(rpc.WithPlugins(func(flavor *rpc.FlavorRPC) []*core.NamedPlugin {
    return []*core.NamedPlugin{{"myplugin1", &MyPlugin{&flavor.GRPC},
                               {"myplugin2", &MyPlugin{&flavor.GRPC},}
  }))
  ```
* if you want to use one simple plugin (without any client or a server) you can write:
  ```
  flavor := &local.FlavorLocal{}
  core.NewAgent(core.Inject(flavor), core.WithPlugin("myplugin1", &MyPlugin{Deps: flavor.PluginInfraDeps("myplugin1")}))
  ```
* if you want to combine multiple flavors to inject their plugins to new MyPlugin
  ```
  loc := &local.FlavorLocal{}
  rpcs := &rpc.FlavorRPC{FlavorLocal: loc}
  cons := &connectors.AllConnectorsFlavor{FlavorLocal: loc}
  core.NewAgent(core.Inject(rpcs, cons), core.WithPlugin("myplugin", &MyPlugin{Deps: Deps{&rpcs.GRPC, &cons.ETCD}}))
  ```

## ETCD/Datasync
* GetPrevValue enabled in the proto watcher API. Etcd-lib/watcher example was updated to show 
  the added functionality.
* Fixed datasync internal state after resync causing that the resynced data were handled as created
  after first modification.
* Fixed issue where datasync plugin was stuck on close

## Cassandra
Added TLS support

# Release v1.0.6 (2017-10-30)

## ETCD/Datasync
* etcd new feature PutIfNotExists adds key-value pair if the key doesn't exist.
* feature GetPrevValue() used to obtain previous value from key-value database was returned to API
* watcher registration object has a new method to close single subscribed key. Key can be un-subscribed in runtime.
  See example usage in [examples/datasync-plugin](examples/datasync-plugin) for more details  

## Documentation
* improved documentation/code comments in datasync, config and core packages 

# Release v1.0.5 (2017-10-17)

## Profiling
* new [logging/measure](logging/measure) - time measurement utility to measure duration of function or 
  any part of the code. Use `NewStopwatch(name string, log logging.Logger)` to create an 
  instance of stopwatch with name, desired logger and table with measured time entries. See 
  [readme](logging/measure/README.md) for more information.

## Kafka
* proto_connection.go and bytes_connection.go consolidated, bytes_connection.go now mirrors all
  the functionality from proto_connection.go.
  * mux can create two types of connection, standard bytes connection and bytes manual connection.
    This enables to call only respective methods on them (to use manual partitioning, it is needed to
    create manual connection, etc.)
  * method ConsumeTopicOnPartition renamed to ConsumeTopic (similar naming as in the proto_connection.go).
    The rest of the signature is not changed.
* post-init watcher enabled in bytes_connection.go api
* added methods MarkOffset and CommitOffsets to both, proto and bytes connection. Automatic offset marking
  was removed
* one instance of mux in kafka plugin
* new field `group-id` can be added to kafka.conf. This value is used as a Group ID in order to set it
  manually. In case the value is not provided, the service label is used instead (just like before).

# Release v1.0.4 (2017-09-25)

## Documentation
* Improved documentation of public APIs (comments)
* Improved documentation of examples (comments, doc.go, README.md)
* Underscore in example suffixes "_lib" and "_plugin" replaced with a dash

## Health, status check & probes
* status check is now registered also for Cassandra & Redis
* new prometheus format probe support (in rpcflavor)

## Profiling
* Logging duration (etcd connection establishment, kafka connection establishment, resync)

## Plugin Configuration
* new [examples/configs-plugin](examples/configs-plugin)
* new flag --config-dir=. (by default "." meaning current working directory)
* configuration files can but not need to have absolute paths anymore (e.g. --kafka-config=kafka.conf)
* if you put all configuration files (etcd.conf, kafka.conf etc.) in one directory agent will load them
* if you want to disable configuration file just put empty value for a particular flag (e.g. --kafka-config)

## Logging
* [logmanager plugin](logging/logmanager)
  * new optional flag --logs-config=logs.conf (showcase in [examples/logs-plugin](examples/logs-plugin))
  * this plugin is now part of LocalFlavor (see field Logs) & tries to load configuration
  * HTTP dependency is optional (if it is not set it just does not registers HTTP handlers)
* logger name added in logger fields (possible to grep only certain logger - effectively by plugin)

## Kafka
* kafka.Plugin.Disabled() returned if there is no kafka.conf present
* Connection in bytes_connection.go renamed to BytesConnection
* kafka plugin initializes two multiplexers for dynamic mode (automatic partitions) and manual mode.
  Every multiplexer can create its own connection and provides access to different set of methods
  (publishing to partition, watching on partition/offset)
* ProtoWatcher from API was changed - methods WatchPartition and StopWatchPartition were removed
  from the ProtoWatcher interface and added to newly created ProtoPartitionWatcher. There is also a new
  method under Mux interface - NewPartitionWatcher(subscriber) which returns ProtoPartitionWatcher
  instance that allows to call partition-related methods
* Offset mark is done for hash/default-partitioned messages only. Manually partitioned message's offset
  is not marked.
* It is possible to start kafka consumer on partition after kafka plugin initialization procedure. New
  example [post-init-consumer](examples/kafka-plugin/post-init-consumer) was created to show the
  functionality
* fixes inside Mux.NewSyncPublisher() & Mux.NewAsyncPublisher() related to previous partition changes
* Known Issues:
  * More than one network connection to Kafka (multiple instances of MUX)
  * TODO Minimalistic examples & documentation for Kafka API will be improved in a later release.

## Flavors
* optionally GPRC server can be enabled in [rpc flavor](flavors/rpc) using --grpc-port=9111 (or using config gprc.conf)
* [Flavor interface](core/list_flavor_plugin.go) now contains three methods: Plugins(), Inject(), LogRegistry() to standardize these methods over all flavors. Note, LogRegistry() is usually embedded using local flavor.

# Release v1.0.3 (2017-09-08)
* [FlavorAllConnectors](flavors/connectors)
    * Inlined plugins: ETCD, Kafka, Redis, Cassandra
* [Kafka Partitions](messaging/kafka)
    * Implemented new methods that allow to specify partitions & offset parameters:
      * publish: Mux.NewSyncPublisherToPartition() & Mux.NewAsyncPublisherToPartition()
      * watch: ProtoWatcher.WatchPartition()
    * Minimalistic examples & documentation for Kafka API will be improved in a later release.

# Release v1.0.2 (2017-08-28)

## Major Themes

The major themes for Release v1.0.2 are as follows:
* Libraries (GO Lang packages) for connecting to Data Bases and Message Bus.
  Set of these libraries provides unified client API and configuration for:
    * [Cassandra](db/sql/cassandra)
    * [etcd](db/keyval/etcd)
    * [Redis](db/keyval/redis)
    * [Kafka](db/)
* [Data Synchronization](datasync) plugin for watching and writing data asynchronously; it is currently implemented only for the [db/keyval API](db/keyval) API. It facilitates reading of data during startup or after reconnection to a data store and then watching incremental changes.
* Agent [Core](core) that provides plugin lifecycle management
(initialization and graceful shutdown of plugins) is able to run
different [flavors](flavors) (reusable collection of plugins):
    * [local flavor](flavors/local) - a minimal collection of plugins:
      * [statuscheck](health/statuscheck)
      * [servicelabel](servicelabel)
      * [resync orch](datasync/restsync)
      * [log registry](logging)
    * [RPC flavor](flavors/rpc) - exposes REST API for all plugins, especially for:
      * [statuscheck](health/statuscheck) (RPCs probed from systems such as K8s)
      * [logging](logging/logmanager) (for changing log level at runtime remotely)
    * connector flavors:
      * Cassandra flavor
      * etcd flavor
      * Redis flavor
      * Kafka flavor
* [Examples](examples)
* [Docker](docker) container-based development environment
* Helpers:
  * [IDX Map](idxmap) is a reusable thread-safe in memory data structure.
      This map is designed for sharing key value based data
      (lookup by primary & secondary indexes, plus watching individual changes).
      It is useful for:
      - implementing backend for plugin specific API shared by multiple plugins;
      - caching of a remote data store.
  * [Config](config): helpers for loading plugin specific configuration.
  * [Service Label](servicelabel): retrieval of a unique identifier for a CN-Infra based app.
