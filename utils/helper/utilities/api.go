package utilities

import "github.com/ligato/networkservicemesh/plugins/idempotent"

type LinkUtilities interface {
	ReadLinkData(link string) (string, error)
}

type PluginAPI interface {
	idempotent.PluginAPI
	LinkUtilities
}
