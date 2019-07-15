package fanout

import (
	"context"
	"github.com/caddyserver/caddy"
	"github.com/coredns/coredns/plugin/pkg/dnstest"
	"github.com/coredns/coredns/plugin/pkg/transport"
	"github.com/coredns/coredns/plugin/test"
	"github.com/miekg/dns"
	"net"
	"sync/atomic"
	"testing"
)

func TestFanout1(t *testing.T) {
	const expected = 2
	i := uint32(0)
	s1 := newServer(func(w dns.ResponseWriter, r *dns.Msg) {
		if r.Question[0].Name == "example.org." {
			msg := dns.Msg{
				Answer: []dns.RR{makeRecordA("example.org. 3600	IN	A 10.0.0.1")},
			}
			atomic.AddUint32(&i, 1)
			msg.SetReply(r)
			w.WriteMsg(&msg)
		}
	}, "example.org.")
	s2 := newServer(func(w dns.ResponseWriter, r *dns.Msg) {
		if r.Question[0].Name == "www.example.org" {
			msg := dns.Msg{
				Answer: []dns.RR{makeRecordA("www.example.org 3600	IN	A 10.0.0.1")},
			}
			atomic.AddUint32(&i, 1)
			msg.SetReply(r)
			w.WriteMsg(&msg)
		}
	}, "www.example.org")
	defer s1.close()
	defer s2.close()

	p1 := newDnsAgent(s1.Addr, transport.DNS)
	p2 := newDnsAgent(s2.Addr, transport.DNS)
	f := NewFanout()
	f.setProxy(p1)
	f.setProxy(p2)
	defer f.Close()

	req := new(dns.Msg)
	req.SetQuestion("example.org.", dns.TypeA)
	f.ServeDNS(context.TODO(), &test.ResponseWriter{}, req)
	req = new(dns.Msg)
	req.SetQuestion("www.example.org", dns.TypeA)
	f.ServeDNS(context.TODO(), &test.ResponseWriter{}, req)
	i1 := atomic.LoadUint32(&i)
	if i1 != expected {
		t.Errorf("Expected number of health checks to be %d, got %d", expected, i1)
	}
}

func TestFanout2(t *testing.T) {
	s := newServer(func(w dns.ResponseWriter, r *dns.Msg) {
		ret := new(dns.Msg)
		ret.SetReply(r)
		ret.Answer = append(ret.Answer, test.A("example.org. IN A 127.0.0.1"))
		w.WriteMsg(ret)
	}, ".")
	defer s.close()
	c := caddy.NewTestController("dns", "fanout "+s.Addr)
	f, err := parseFanout(c)
	if err != nil {
		t.Errorf("Failed to create fanout: %s", err)
	}
	f.OnStartup()
	defer f.OnShutdown()

	m := new(dns.Msg)
	m.SetQuestion("example.org.", dns.TypeA)
	rec := dnstest.NewRecorder(&test.ResponseWriter{})

	if _, err := f.ServeDNS(context.TODO(), rec, m); err != nil {
		t.Fatal("Expected to receive reply, but didn't")
	}
	if x := rec.Msg.Answer[0].Header().Name; x != "example.org." {
		t.Errorf("Expected %s, got %s", "example.org.", x)
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

	for i := 0; i < retryCount; i++ {
		s1.Listener, _ = net.Listen("tcp", ":0")
		if s1.Listener != nil {
			break
		}
		s1.Listener.Close()
		s1.Listener = nil
	}
	if s1.Listener == nil {
		panic("failed to create new server")
	}

	s1.NotifyStartedFunc = func() { close(ch1) }
	go s1.ActivateAndServe()

	<-ch1
	return &server{inner: s1, Addr: s1.Listener.Addr().String()}
}

func makeRecordA(rr string) *dns.A {
	r, _ := dns.NewRR(rr);
	return r.(*dns.A)
}
