package mechanism_kernel

import "os"

const (
	ForwarderAllowVHost = "FORWARDER_ALLOW_VHOST" // To disallow VHOST please pass "false" into this env variable.
)

func useVHostNet() bool {
	vhostAllowed := os.Getenv(ForwarderAllowVHost)
	if vhostAllowed == "false" {
		return false
	}
	if _, err := os.Stat("/dev/vhost-net"); err == nil {
		return true
	}
	return false
}
