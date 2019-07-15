package fanout

import (
	"context"
	"testing"

	"github.com/coredns/coredns/plugin/pkg/dnstest"
	"github.com/coredns/coredns/plugin/pkg/transport"
	"github.com/coredns/coredns/plugin/test"
	"github.com/coredns/coredns/request"

	"github.com/miekg/dns"
)

func TestDnsAgentClose(t *testing.T) {
	s := dnstest.NewServer(func(w dns.ResponseWriter, r *dns.Msg) {
		ret := new(dns.Msg)
		ret.SetReply(r)
		w.WriteMsg(ret)
	})
	defer s.Close()

	req := new(dns.Msg)
	req.SetQuestion("example.org.", dns.TypeA)
	state := request.Request{W: &test.ResponseWriter{}, Req: req}
	ctx := context.TODO()

	for i := 0; i < 100; i++ {
		p := newDnsAgent(s.Addr, transport.DNS)
		p.start(healthClientInterval)
		go func() { p.Connect(ctx, state) }()
		go func() { p.Connect(ctx, state) }()

		p.close()
	}
}

func TestProtocolSelection(t *testing.T) {
	p := newDnsAgent("bad_address", transport.DNS)

	stateTCP := request.Request{W: &test.ResponseWriter{TCP: true}, Req: new(dns.Msg)}
	ctx := context.TODO()

	go func() {
		p.Connect(ctx, stateTCP)
	}()

	for i, exp := range []string{"tcp"} {
		proto := <-p.transport.dial
		p.transport.ret <- nil
		if proto != exp {
			t.Errorf("Unexpected protocol in case %d, expected %q, actual %q", i, exp, proto)
		}
	}
}
