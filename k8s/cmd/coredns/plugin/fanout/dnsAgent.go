package fanout

import (
	"crypto/tls"
	"runtime"
	"time"

	"github.com/coredns/coredns/plugin/pkg/up"
)

type dnsAgent struct {
	addr      string
	expire    time.Duration
	transport *Transport
	probe     *up.Probe
	health    HealthChecker
}

func newDnsAgent(addr, trans string) *dnsAgent {
	a := &dnsAgent{
		addr:      addr,
		probe:     up.New(),
		transport: newTransport(addr),
	}
	a.health = NewHealthChecker(trans)
	runtime.SetFinalizer(a, (*dnsAgent).finalizer)
	return a
}

func (p *dnsAgent) setTLSConfig(cfg *tls.Config) {
	p.transport.setTLSConfig(cfg)
	p.health.SetTLSConfig(cfg)
}

func (p *dnsAgent) setExpire(expire time.Duration) {
	p.transport.setExpire(expire)
}

func (p *dnsAgent) healthcheck() {
	if p.health == nil {
		log.Warning("No healthchecker")
		return
	}

	p.probe.Do(func() error {
		return p.health.Check(p)
	})
}

func (p *dnsAgent) close() {
	p.probe.Stop()
}
func (p *dnsAgent) finalizer() {
	p.transport.Stop()
}
func (p *dnsAgent) start(duration time.Duration) {
	p.probe.Start(duration)
	p.transport.Start()
}
