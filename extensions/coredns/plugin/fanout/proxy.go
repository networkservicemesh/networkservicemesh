package fanout

import (
	"crypto/tls"
	"runtime"
	"time"

	"github.com/coredns/coredns/plugin/pkg/up"
)

// DnsServerDefinition defines an upstream host.
type DnsServerDefinition struct {
	addr string

	// Connection caching
	expire    time.Duration
	transport *Transport

	// health checking
	probe  *up.Probe
	health HealthChecker
}

// NewDNSServerDefinition returns a new proxy.
func NewDNSServerDefinition(addr, trans string) *DnsServerDefinition {
	p := &DnsServerDefinition{
		addr:      addr,
		probe:     up.New(),
		transport: newTransport(addr),
	}
	p.health = NewHealthChecker(trans)
	runtime.SetFinalizer(p, (*DnsServerDefinition).finalizer)
	return p
}

// SetTLSConfig sets the TLS config in the lower p.transport and in the healthchecking client.
func (p *DnsServerDefinition) SetTLSConfig(cfg *tls.Config) {
	p.transport.SetTLSConfig(cfg)
	p.health.SetTLSConfig(cfg)
}

// SetExpire sets the expire duration in the lower p.transport.
func (p *DnsServerDefinition) SetExpire(expire time.Duration) { p.transport.SetExpire(expire) }

// Healthcheck kicks of a round of health checks for this proxy.
func (p *DnsServerDefinition) Healthcheck() {
	if p.health == nil {
		log.Warning("No healthchecker")
		return
	}

	p.probe.Do(func() error {
		return p.health.Check(p)
	})
}

// close stops the health checking goroutine.
func (p *DnsServerDefinition) close()     { p.probe.Stop() }
func (p *DnsServerDefinition) finalizer() { p.transport.Stop() }

// start starts the proxy's healthchecking.
func (p *DnsServerDefinition) start(duration time.Duration) {
	p.probe.Start(duration)
	p.transport.Start()
}

const (
	maxTimeout = 2 * time.Second
	hcInterval = 500 * time.Millisecond
)
