package state

import (
	"context"

	"github.com/ligato/vpp-agent/api/configurator"
)

type dataChangeState string

const (
	dataChangeKey dataChangeState = "dataChange"
)

func WithDataChange(ctx context.Context, dataChange *configurator.Config) context.Context {
	return context.WithValue(ctx, dataChangeKey, dataChange)
}

func DataChange(ctx context.Context) *configurator.Config {
	v := ctx.Value(dataChangeKey)
	if v == nil {
		return nil
	}
	if dataChange, ok := v.(*configurator.Config); ok {
		return dataChange
	}
	return nil
}
