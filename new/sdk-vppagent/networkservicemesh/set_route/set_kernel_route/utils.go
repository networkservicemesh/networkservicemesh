package set_kernel_route

import "net"

func extractCleanIPAddress(addr string) string {
	ip, _, err := net.ParseCIDR(addr)
	if err == nil {
		return ip.String()
	}
	return addr
}
