package env

import "github.com/networkservicemesh/networkservicemesh/utils"

const (
	//UseUpdateAPIEnv means need to start update dns context server
	UseUpdateAPIEnv = utils.EnvVar("USE_UPDATE_API")
	//UpdateAPIClientSock means path to client socket for dns context update server
	UpdateAPIClientSock = utils.EnvVar("UPDATE_API_CLIENT_SOCKET")
	//DefaultDNSServerIPList using for configuring default config
	DefaultDNSServerIPList = utils.EnvVar("UPDATE_API_DEFAULT_DNS_SERVER")
)
