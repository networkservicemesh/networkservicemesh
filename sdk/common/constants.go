package common

const (
	// NsmServerSocketEnv is the name of the env variable to define NSM server socket
	NsmServerSocketEnv = "NSM_SERVER_SOCKET"
	// NsmClientSocketEnv is the name of the env variable to define NSM client socket
	NsmClientSocketEnv = "NSM_CLIENT_SOCKET"
	// WorkspaceEnv is the name of the env variable to define workspace directory
	WorkspaceEnv = "WORKSPACE"
	// NsmWorkspaceTokenEnv the name of the env variable with a token to identify client in NSM
	NsmWorkspaceTokenEnv = "NSM_WORKSPACE_TOKEN"
	// WorkspaceTokenHeader the header to pass workspace token to server side of GRPC call
	WorkspaceTokenHeader = "nsm-workspace-token"
)
