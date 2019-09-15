package main

import (
	"github.com/networkservicemesh/networkservicemesh/utils"
)

const (
	//SearchDomainsEnv means domains for dnsConfig. It used only with flag -dns
	SearchDomainsEnv utils.EnvVar = "DNS_SEARCH_DOMAINS"
	//ServerIPsEnv means dns server ips for dnsConfig. It used only with flag -dns
	ServerIPsEnv utils.EnvVar = "DNS_SERVER_IPS"
	//SrcMacAddr means source mac address
	SrcMacAddr utils.EnvVar = "SRC_MAC_ADDR"
	//DstMacAddr means destination mac address
	DstMacAddr utils.EnvVar = "DST_MAC_ADDR"
)
