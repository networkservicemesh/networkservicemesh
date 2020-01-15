package vppagent

import (
	"context"

	"github.com/ligato/vpp-agent/api/configurator"
	"github.com/ligato/vpp-agent/api/models/linux"
	"github.com/ligato/vpp-agent/api/models/netalloc"
	"github.com/ligato/vpp-agent/api/models/vpp"
)

type contextKeyType string

const (
	configKey contextKeyType = "configKey"
)

//WithConfig gets vppagent config from context
func withConfig(ctx context.Context) context.Context {
	if config, ok := ctx.Value(configKey).(*configurator.Config); ok && config != nil {
		return ctx
	}
	rv := &configurator.Config{
		VppConfig:      &vpp.ConfigData{},
		LinuxConfig:    &linux.ConfigData{},
		NetallocConfig: &netalloc.ConfigData{},
	}
	return context.WithValue(ctx, configKey, rv)
}

func Config(ctx context.Context) *configurator.Config {
	if rv, ok := ctx.Value(configKey).(*configurator.Config); ok {
		return rv
	}
	return nil
}
