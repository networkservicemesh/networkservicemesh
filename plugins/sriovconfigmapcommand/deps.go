package sriovconfigmapcommand

import (
	"github.com/ligato/networkservicemesh/plugins/k8sclient"
	"github.com/ligato/networkservicemesh/plugins/logger"
	"github.com/ligato/networkservicemesh/utils/helper/utilities"
)

// Deps defines dependencies of sriov config map command plugin.
type Deps struct {
	Name      string
	Log       logger.FieldLoggerPlugin
	Utilities utilities.PluginAPI
	K8sClient k8sclient.PluginAPI
}
