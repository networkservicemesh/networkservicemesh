package kernel

const (
	MECHANISM = "KERNEL_INTERFACE"

	// Parameters

	// LinuxIfMaxLength - Linux has a limit of 15 characters for an interface name
	LinuxIfMaxLength = 15
	// SocketFilename - socket filename memif mechanism property key
	SocketFilename = "socketfile"
	// Master - NSMgr name
	Master = "master"
	// Slave - NSMgr name
	Slave = "slave"
	// WorkspaceNSEName - NSE workspace name mechanism property key
	WorkspaceNSEName = "workspaceNseName"
	// MemifSocket - memif socket filename
	MemifSocket = "memif.sock"
	// NsmBaseDirEnv - NSM location directory
	NsmBaseDirEnv = "NSM_BASE_DIR"
)
