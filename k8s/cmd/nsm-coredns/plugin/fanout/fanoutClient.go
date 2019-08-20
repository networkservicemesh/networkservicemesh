package fanout

import (
	"crypto/tls"
	"runtime"
	"time"

	"github.com/coredns/coredns/plugin/pkg/up"
	"github.com/pkg/errors"
)

type fanoutClient struct {
	addr      string
	transport *Transport
	health    HealthChecker
}

func createFanoutClient(addr string) *fanoutClient {
	P := up.New()
	P.Start(time.Millisecond)

	a := &fanoutClient{
		addr:      addr,
		transport: newTransport(addr),
		health:    NewHealthChecker(addr),
	}
	runtime.SetFinalizer(a, (*fanoutClient).finalizer)
	return a
}

func (p *fanoutClient) setTLSConfig(cfg *tls.Config) {
	p.transport.setTLSConfig(cfg)
	p.health.SetTLSConfig(cfg)
}

func (p *fanoutClient) setExpire(expire time.Duration) {
	p.transport.setExpire(expire)
}

func (p *fanoutClient) healthCheck() error {
	if p.health == nil {
		return errors.New("no healthchecker")
	}
	return p.health.Check()
}

func (p *fanoutClient) finalizer() {
	p.transport.Stop()
}

func (p *fanoutClient) start() {
	p.transport.Start()
}
