package kernel

const (
	Mechanism = "KERNEL_INTERFACE"

	// Parameters

	// NetNsInodeKey - netns inode mechanism property key
	NetNsInodeKey = "netnsInode"
	// InterfaceNameKey - interface name mechanism property key
	InterfaceNameKey = "name"
	// InterfaceDescriptionKey - interface description mechanism property key
	InterfaceDescriptionKey = "description"
	// LinuxIfMaxLength - Linux has a limit of 15 characters for an interface name
	LinuxIfMaxLength = 15
	// SocketFilename - socket filename memif mechanism property key
	SocketFilename = "socketfile"
	// Master - NSMgr name
	Master = "master"
	// Slave - NSMgr name
	Slave = "slave"
	// Workspace - NSM workspace location mechanism property key
	Workspace = "workspace"
	// WorkspaceNSEName - NSE workspace name mechanism property key
	WorkspaceNSEName = "workspaceNseName"
	// MemifSocket - memif socket filename
	MemifSocket = "memif.sock"
	// NsmBaseDirEnv - NSM location directory
	NsmBaseDirEnv = "NSM_BASE_DIR"
)
