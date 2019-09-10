package dataplane

import (
	"context"

	"github.com/ligato/vpp-agent/api/configurator"
)

type dataChangeKeyType string

const (
	dataChangeKey dataChangeKeyType = "dataChange"
)

//WithDataChange put dataChange config into context
func WithDataChange(ctx context.Context, dataChange *configurator.Config) context.Context {
	return context.WithValue(ctx, dataChangeKey, dataChange)
}

//DataChange gets dataChange config from context
func DataChange(ctx context.Context) *configurator.Config {
	if dataChange, ok := ctx.Value(dataChangeKey).(*configurator.Config); ok {
		return dataChange
	}
	return nil
}
