package client_url

import (
	"context"
	"net/url"
)

const (
	clientUrlKey contextKeyType = "ClientUrl"
)

type contextKeyType string

// WithClientUrl -
//    Wraps 'parent' in a new Context that has the ClientUrl
func WithClientUrl(parent context.Context, clientUrl *url.URL) context.Context {
	if parent == nil {
		parent = context.TODO()
	}
	return context.WithValue(parent, clientUrlKey, clientUrl)
}

// ClientUrl -
//   Returns the ClientUrl
func ClientUrl(ctx context.Context) *url.URL {
	if rv, ok := ctx.Value(clientUrlKey).(*url.URL); ok {
		return rv
	}
	return nil
}
