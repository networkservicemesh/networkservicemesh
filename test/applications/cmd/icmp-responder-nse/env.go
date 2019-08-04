package main

import "github.com/networkservicemesh/networkservicemesh/sdk/common"

const (
	//SearchDomainsEnv means domains for dnsConfig. It used only with flag -dns
	SearchDomainsEnv common.EnvVar = "DNS_SEARCH_DOMAINS"
	//ServerIPsEnv means dns server ips for dnsConfig. It used only with flag -dns
	ServerIPsEnv common.EnvVar = "DNS_SERVER_IPS"
)
