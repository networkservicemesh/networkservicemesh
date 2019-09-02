package connection

const (
	NetNsInodeKey           = "netnsInode"
	PodNameKey              = "podName"
	InterfaceNameKey        = "name"
	InterfaceDescriptionKey = "description"
	LinuxIfMaxLength        = 15 // Linux has a limit of 15 characters for an interface name
	SocketFilename          = "socketfile"
	Master                  = "master"
	Slave                   = "slave"
	Workspace               = "workspace"
	WorkspaceNSEName        = "workspaceNseName"
	MemifSocket             = "memif.sock"
	NsmBaseDirEnv           = "NSM_BASE_DIR"
)
