package tools

import (
	. "context"

	. "google.golang.org/grpc/metadata"
)

// MetadataWithIncomingContext - Creates new context with incoming md attached.
func MetadataWithIncomingContext(srcCtx, incomingCtx Context) Context {
	if incomingMd, ok := FromIncomingContext(incomingCtx); ok {
		md := incomingMd
		if srcMd, ok := FromOutgoingContext(srcCtx); ok {
			md = Join(md, srcMd)
		}
		return NewOutgoingContext(srcCtx, md)
	}
	return srcCtx
}

// MetadataWithPair - Creates new context with joined single MD (joined multiple values by a key)
func MetadataWithPair(srcCtx Context, kv ...string) Context {
	md := Pairs(kv...)
	if srcMd, ok := FromOutgoingContext(srcCtx); ok {
		md = Join(md, srcMd)
	}
	return NewOutgoingContext(srcCtx, md)
}

// MetadataFromIncomingContext - obtains the values from context for a given key.
func MetadataFromIncomingContext(ctx Context, key string) []string {
	if md, ok := FromIncomingContext(ctx); ok {
		return md.Get(key)
	}
	return []string{}
}
