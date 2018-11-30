package utils

import "strings"

func parseNetworkService(networkService string) (string, string) {
	keyValue := strings.Split(networkService, ":")
	return strings.Trim(keyValue[0], " "), strings.Trim(keyValue[1], " ")
}

func ParseNetworkServices(nsConfig string) map[string]string {
	configMap := make(map[string]string)
	networkServices := strings.Split(nsConfig, ",")
	for _, pair := range networkServices {
		intf, nsName := parseNetworkService(pair)
		configMap[nsName] = intf
	}
	return configMap
}
