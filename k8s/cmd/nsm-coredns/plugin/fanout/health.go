package fanout

import (
	"crypto/tls"
	"time"

	"github.com/miekg/dns"
)

// HealthChecker checks the upstream health.
type HealthChecker interface {
	//Check is used as the up.Func in the up.Probe.
	Check() error
	//SetTLSConfig sets tls config for checker.
	SetTLSConfig(*tls.Config)
}

type dnsHealthClient struct {
	c    *dns.Client
	addr string
}

// NewHealthChecker returns a new HealthChecker based on Transport.
func NewHealthChecker(addr string) HealthChecker {
	c := new(dns.Client)
	c.Net = "tcp"
	c.ReadTimeout = 1 * time.Second
	c.WriteTimeout = 1 * time.Second

	return &dnsHealthClient{c: c, addr: addr}
}

func (h *dnsHealthClient) SetTLSConfig(cfg *tls.Config) {
	h.c.Net = tcptlc
	h.c.TLSConfig = cfg
}

func (h *dnsHealthClient) Check() error {
	err := h.askAny(h.addr)
	if err != nil {
		return err
	}
	return nil
}

func (h *dnsHealthClient) askAny(addr string) error {
	ping := new(dns.Msg)
	ping.SetQuestion(".", dns.TypeNS)
	m, _, err := h.c.Exchange(ping, addr)
	if err != nil && m != nil {
		if m.Response || m.Opcode == dns.OpcodeQuery {
			err = nil
		}
	}
	return err
}
