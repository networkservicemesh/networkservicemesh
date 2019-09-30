package tests

import "github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"

func newTestNse(name, networkServiceName string) *registry.NSERegistration {
	return newTestNseWithPayload(name, networkServiceName, "IP")
}
func newTestNseWithPayload(name, networkServiceName, payload string) *registry.NSERegistration {
	return &registry.NSERegistration{
		NetworkService: &registry.NetworkService{
			Name:    networkServiceName,
			Payload: payload,
		},
		NetworkServiceManager: &registry.NetworkServiceManager{},
		NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
			Name:    name,
			Payload: payload,
		},
	}
}
