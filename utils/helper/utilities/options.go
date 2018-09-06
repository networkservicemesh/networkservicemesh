package utilities

import (
	"github.com/ligato/networkservicemesh/plugins/logger/hooks/pid"
	"github.com/ligato/networkservicemesh/utils/registry"
	"github.com/onrik/logrus/filename"
	"github.com/sirupsen/logrus"
)

// Option acts on a Plugin in order to set its Deps or Config
type Option func(*Plugin)

// NewPlugin creates a new Plugin with Deps/Config set by the supplied opts
func NewPlugin(opts ...Option) *Plugin {
	p := &Plugin{}
	for _, o := range opts {
		o(p)
	}
	DefaultDeps()(p)
	return p
}

// SharedPlugin provides a single shared Plugin that has the same Deps/Config as would result
// from the application of opts
func SharedPlugin(opts ...Option) *Plugin {
	p := NewPlugin(opts...)
	return registry.Shared().LoadOrStore(p).(*Plugin)
}

// UseDeps creates an Option to set the Deps for a Plugin
func UseDeps(deps *Deps) Option {
	return func(p *Plugin) {
	}
}

var defaultHooks = []logrus.Hook{filename.NewHook(), pid.NewHook()}

// DefaultDeps creates an Option to set any unset Dependencies to Default Values
// DefaultDeps() is always applied by NewPlugin/SharedPlugin after all other Options
func DefaultDeps() Option {
	return func(p *Plugin) {
	}
}
