package env

import "github.com/networkservicemesh/networkservicemesh/utils"

const (
	//UseUpdateApiEnv means need to start update dns context server
	UseUpdateApiEnv = utils.EnvVar("USE_UPDATE_API")
	//UpdateAPIClientSock means path to client socket for dns context update server
	UpdateAPIClientSock = utils.EnvVar("UPDATE_API_CLIENT_SOCKET")
	//DefaultDNSServerIP using for configuring default config
	DefaultDNSServerIP = utils.EnvVar("UPDATE_API_DEFAULT_DNS_SERVER")
)
