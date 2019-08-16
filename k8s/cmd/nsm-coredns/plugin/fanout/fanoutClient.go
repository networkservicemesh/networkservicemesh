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

func (p *fanoutClient) healthCheck(maxFails int) error {
	if p.health == nil {
		return errors.New("no healthchecker")
	}
	err := error(nil)
	for i := 0; i < maxFails; i++ {
		checkErr := p.health.Check()
		if checkErr == nil {
			return nil
		}
		if err != nil {
			err = errors.Wrap(err, checkErr.Error())
		} else {
			err = checkErr
		}
	}
	return err
}

func (p *fanoutClient) finalizer() {
	p.transport.Stop()
}

func (p *fanoutClient) start() {
	p.transport.Start()
}
