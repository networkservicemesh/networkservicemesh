package main

import "github.com/networkservicemesh/networkservicemesh/utils"

const (
	//MonitorDNSConfigsEnv means boolean flag. If the flag is true then nsm-monitor will monitor DNS configs
	MonitorDNSConfigsEnv utils.EnvVar = "MONITOR_DNS_CONFIGS_ENV"
	//PathToCorefileEnv means path to corefile
	PathToCorefileEnv utils.EnvVar = "PATH_TO_COREFILE"
	//ReloadCorefileEnvTime means time to reload corefile
	ReloadCorefilePeriodEnv utils.EnvVar = "RELOAD_COREFILE_TIME"
)
