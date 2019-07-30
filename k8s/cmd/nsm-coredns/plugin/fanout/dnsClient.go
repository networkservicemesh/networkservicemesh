package fanout

import (
	"crypto/tls"
	"runtime"
	"time"

	"github.com/coredns/coredns/plugin/pkg/up"
)

type dnsClient struct {
	addr      string
	transport *Transport
	probe     *up.Probe
	health    HealthChecker
}

func createDNSClient(addr, trans string) *dnsClient {
	a := &dnsClient{
		addr:      addr,
		probe:     up.New(),
		transport: newTransport(addr),
	}
	a.health = NewHealthChecker(trans)
	runtime.SetFinalizer(a, (*dnsClient).finalizer)
	return a
}

func (p *dnsClient) setTLSConfig(cfg *tls.Config) {
	p.transport.setTLSConfig(cfg)
	p.health.SetTLSConfig(cfg)
}

func (p *dnsClient) setExpire(expire time.Duration) {
	p.transport.setExpire(expire)
}

func (p *dnsClient) healthCheck() {
	if p.health == nil {
		log.Warning("No healthchecker")
		return
	}

	p.probe.Do(func() error {
		return p.health.Check(p)
	})
}

func (p *dnsClient) close() {
	p.probe.Stop()
}
func (p *dnsClient) finalizer() {
	p.transport.Stop()
}
func (p *dnsClient) start(duration time.Duration) {
	p.probe.Start(duration)
	p.transport.Start()
}
