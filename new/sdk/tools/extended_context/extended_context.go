package extended_context

import "context"

type extendedContext struct {
	context.Context
	valuesContext context.Context
}

func (ec *extendedContext) Value(key interface{}) interface{} {
	return ec.valuesContext.Value(key)
}

func New(ctx context.Context, valuesContext context.Context) context.Context {
	return &extendedContext{
		Context:       ctx,
		valuesContext: valuesContext,
	}
}
