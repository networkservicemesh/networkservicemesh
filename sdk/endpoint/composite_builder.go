package endpoint

//CompositeEndpointBuilder - Builder interface for building CompositeEndpoint
type CompositeEndpointBuilder interface {
	Append(...ChainedEndpoint) CompositeEndpointBuilder
	Build() *CompositeEndpoint
}

//NewCompositeEndpointBuilder - Creates instance of CompositeEndpointBuilder
func NewCompositeEndpointBuilder() CompositeEndpointBuilder {
	return &compositeEndpointBuilder{
		endpoints: make([]ChainedEndpoint, 0, 4),
	}
}

type compositeEndpointBuilder struct {
	endpoints []ChainedEndpoint
}

//Append - Appends ChainedEndpoint to chain of endpoints
func (c *compositeEndpointBuilder) Append(eps ...ChainedEndpoint) CompositeEndpointBuilder {
	c.endpoints = append(c.endpoints, eps...)
	return c
}

//Build - Builds instance of CompositeEndpoint
func (c *compositeEndpointBuilder) Build() *CompositeEndpoint {
	return NewCompositeEndpoint(c.endpoints...)
}
