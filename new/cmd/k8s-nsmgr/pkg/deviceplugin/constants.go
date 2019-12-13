package deviceplugin

const (
	// A number of devices we have in buffer for use, so we hold extra DeviceBuffer count of deviceids send to kubelet.
	DeviceBuffer = 30
	//TODO - look at moving the BaseDir to constants somewhere in SDK
	BaseDir     = "/var/lib/networkservicemesh/"
	SpireSocket = "/run/spire/sockets"
)
