package registry

// EndpointNSMName -  - a type to hold endpoint and nsm url composite type.
type EndpointNSMName string

//GetEndpointNSMName - return a Endpoint.Name + ":" + NetworkServiceManager.Url
func (nse *NSERegistration) GetEndpointNSMName() EndpointNSMName {
	if nse == nil {
		return ""
	}
	return NewEndpointNSMName(nse.NetworkServiceEndpoint, nse.NetworkServiceManager)
}

//NewEndpointNSMName - construct an NewEndpointNSMName from endpoint and manager
func NewEndpointNSMName(endpoint *NetworkServiceEndpoint, manager *NetworkServiceManager) EndpointNSMName {
	return EndpointNSMName(endpoint.Name + ":" + manager.Name + "@" + manager.Url)
}
