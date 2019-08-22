package flags

import "flag"

//ICMPResponderFlags represents flags for test application icmp-responder-nse
type ICMPResponderFlags struct {
	//Dirty means not delete itself from registry at the end
	Dirty bool
	//Neighbors means set all available IpNeighbors to connection.Context
	Neighbors bool
	//Routes means set route 8.8.8.8/30 to connection.Context
	Routes bool
	//Update means set route 8.8.8.8/30 to connection.Context
	Update bool
	//DNS means use DNS mutator to update connection. For customize DNSConfig use env variables: DNS_SEARCH_DOMAINS, DNS_SERVER_IPS
	DNS bool
}

//Commands converts flags to strings format
func (f ICMPResponderFlags) Commands() []string {
	commands := []string{"/bin/icmp-responder-nse"}

	if f.Dirty {
		commands = append(commands, "-dirty")
	}
	if f.Neighbors {
		commands = append(commands, "-neighbors")
	}
	if f.Routes {
		commands = append(commands, "-routes")
	}
	if f.Update {
		commands = append(commands, "-update")
	}
	if f.DNS {
		commands = append(commands, "-dns")
	}
	return commands
}

//ParseFlags parses icmp-responder-nse flags
func ParseFlags() ICMPResponderFlags {
	dirty := flag.Bool("dirty", false,
		"will not delete itself from registry at the end")
	neighbors := flag.Bool("neighbors", false,
		"will set all available IpNeighbors to connection.Context")
	routes := flag.Bool("routes", false,
		"will set route 8.8.8.8/30 to connection.Context")
	update := flag.Bool("update", false,
		"will send update to local.Connection after some time")
	dns := flag.Bool("dns", false,
		"use DNS mutator to update connection. For customize DNSConfig use env variables: DNS_SEARCH_DOMAINS, DNS_SERVER_IPS")

	flag.Parse()
	return ICMPResponderFlags{
		Dirty:     *dirty,
		Neighbors: *neighbors,
		Routes:    *routes,
		Update:    *update,
		DNS:       *dns,
	}
}
