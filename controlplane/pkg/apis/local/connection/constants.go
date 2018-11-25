package connection

const (
	NetNsInodeKey           = "netnsInode"
	InterfaceNameKey        = "name"
	InterfaceDescriptionKey = "description"
	LinuxIfMaxLength        = 15 // Linux has a limit of 15 characters for an interface name
	SocketFilename          = "socketfile"
	Master                  = "master"
	Slave                   = "slave"
	Workspace               = "workspace"
	MemifSocket             = "memif.sock"
	NsmBaseDirEnv           = "NSM_BASE_DIR"
)
