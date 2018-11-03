package main

import (
	"github.com/ligato/cn-infra/datasync"
	"github.com/ligato/cn-infra/flavors/local"
)

// Deps lists dependencies of ExamplePlugin.
type Deps struct {
	local.PluginInfraDeps                             // injected
	Publisher             datasync.KeyProtoValWriter  // injected - To write ETCD data
	Watcher               datasync.KeyValProtoWatcher // injected - To watch ETCD data
}
