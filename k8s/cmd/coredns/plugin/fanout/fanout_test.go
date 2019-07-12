package fanout

import (
	"context"
	"github.com/coredns/coredns/plugin/pkg/transport"
	"github.com/coredns/coredns/plugin/test"
	"github.com/miekg/dns"
	"net"
	"sync/atomic"
	"testing"
	"time"
)

func TestFanout(t *testing.T) {
	const expected = 2
	i := uint32(0)
	s1 := newServer(func(w dns.ResponseWriter, r *dns.Msg) {
		if r.Question[0].Name == "example.org." {
			msg := dns.Msg{
				Answer: []dns.RR{A("example.org. 3600	IN	A 10.0.0.1")},
			}
			atomic.AddUint32(&i, 1)
			msg.SetReply(r)
			w.WriteMsg(&msg)
		}
	}, "example.org.")
	s2 := newServer(func(w dns.ResponseWriter, r *dns.Msg) {
		if r.Question[0].Name == "www.example.org" {
			msg := dns.Msg{
				Answer: []dns.RR{A("www.example.org 3600	IN	A 10.0.0.1")},
			}
			atomic.AddUint32(&i, 1)
			msg.SetReply(r)
			w.WriteMsg(&msg)
		}
	}, "www.example.org")
	defer s1.close()
	defer s2.close()

	p1 := NewDNSServerDefinition(s1.Addr, transport.DNS)
	p2 := NewDNSServerDefinition(s2.Addr, transport.DNS)
	f := New()
	f.SetProxy(p1)
	f.SetProxy(p2)
	defer f.Close()

	req := new(dns.Msg)
	req.SetQuestion("example.org.", dns.TypeA)
	f.ServeDNS(context.TODO(), &test.ResponseWriter{}, req)
	time.Sleep(5 * time.Second)
	req = new(dns.Msg)
	req.SetQuestion("www.example.org", dns.TypeA)
	f.ServeDNS(context.TODO(), &test.ResponseWriter{}, req)

	time.Sleep(5 * time.Second)
	i1 := atomic.LoadUint32(&i)
	if i1 != expected {
		t.Errorf("Expected number of health checks to be %d, got %d", expected, i1)
	}
}

type server struct {
	Addr  string
	inner *dns.Server
}

func (s *server) close() {
	s.inner.Shutdown()
}

func newServer(f dns.HandlerFunc, zone string) *server {
	dns.HandleFunc(zone, f)

	ch1 := make(chan bool)

	s1 := &dns.Server{}

	for i := 0; i < 5; i++ { // 5 attempts
		s1.Listener, _ = net.Listen("tcp", ":0")
		if s1.Listener != nil {
			break
		}
		s1.Listener.Close()
		s1.Listener = nil
	}
	if s1.Listener == nil {
		panic("dnstest.NewServer(): failed to create new server")
	}

	s1.NotifyStartedFunc = func() { close(ch1) }
	go s1.ActivateAndServe()

	<-ch1
	return &server{inner: s1, Addr: s1.Listener.Addr().String()}
}

// A returns an A record from rr. It panics on errors.
func A(rr string) *dns.A { r, _ := dns.NewRR(rr); return r.(*dns.A) }
