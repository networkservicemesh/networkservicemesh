package sriovconfigmapcommand

import "github.com/ligato/networkservicemesh/plugins/idempotent"

type PluginAPI interface {
	idempotent.PluginAPI
	Run() error
}
