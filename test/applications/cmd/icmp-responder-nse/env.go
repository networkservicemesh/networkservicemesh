package main

import (
	"github.com/networkservicemesh/networkservicemesh/utils"
)

const (
	//SearchDomainsEnv means domains for dnsConfig. It used only with flag -dns
	SearchDomainsEnv utils.EnvVar = "DNS_SEARCH_DOMAINS"
	//ServerIPsEnv means dns server ips for dnsConfig. It used only with flag -dns
	ServerIPsEnv utils.EnvVar = "DNS_SERVER_IPS"
	
)
