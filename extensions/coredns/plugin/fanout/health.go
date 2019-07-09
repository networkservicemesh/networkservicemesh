package fanout

import (
	"crypto/tls"
	"time"

	"github.com/coredns/coredns/plugin/pkg/transport"

	"github.com/miekg/dns"
)

// HealthChecker checks the upstream health.
type HealthChecker interface {
	Check(*DnsServerDefinition) error
	SetTLSConfig(*tls.Config)
}

// dnsHc is a health checker for a DNS endpoint (DNS, and DoT).
type dnsHc struct{ c *dns.Client }

// NewHealthChecker returns a new HealthChecker based on transport.
func NewHealthChecker(trans string) HealthChecker {
	switch trans {
	case transport.DNS, transport.TLS:
		c := new(dns.Client)
		c.Net = "udp"
		c.ReadTimeout = 1 * time.Second
		c.WriteTimeout = 1 * time.Second

		return &dnsHc{c: c}
	}

	log.Warningf("No healthchecker for transport %q", trans)
	return nil
}

func (h *dnsHc) SetTLSConfig(cfg *tls.Config) {
	h.c.Net = "tcp-tls"
	h.c.TLSConfig = cfg
}

// Check is used as the up.Func in the up.Probe.
func (h *dnsHc) Check(p *DnsServerDefinition) error {
	err := h.send(p.addr)
	if err != nil {
		return err
	}
	return nil
}

func (h *dnsHc) send(addr string) error {
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
