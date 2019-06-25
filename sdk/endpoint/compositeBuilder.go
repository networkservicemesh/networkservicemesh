package endpoint

type CompositeEndpointBuilder interface {
	Append(...ChainedEndpoint) CompositeEndpointBuilder
	Build() *CompositeEndpoint
}

func NewCompositeEndpointBuilder() CompositeEndpointBuilder {
	return &compositeEndpointBuilder{
		endpoints: make([]ChainedEndpoint, 0, 4),
	}
}

type compositeEndpointBuilder struct {
	endpoints []ChainedEndpoint
}

func (c *compositeEndpointBuilder) Append(eps ...ChainedEndpoint) CompositeEndpointBuilder {
	c.endpoints = append(c.endpoints, eps...)
	return c
}
func (c *compositeEndpointBuilder) Build() *CompositeEndpoint {
	return NewCompositeEndpoint(c.endpoints...)
}
