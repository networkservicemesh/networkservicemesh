package fanout

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/caddyserver/caddy"
	"github.com/coredns/coredns/plugin/pkg/dnstest"
	"github.com/coredns/coredns/plugin/pkg/transport"
	"github.com/coredns/coredns/plugin/test"
	"github.com/miekg/dns"
)

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

	for i := 0; i < 10; i++ {
		s1.Listener, _ = net.Listen("tcp", ":0")
		if s1.Listener != nil {
			break
		}
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
	r, _ := dns.NewRR(rr)
	return r.(*dns.A)
}

func TestFanoutTwoServers(t *testing.T) {
	const expected = 2
	answerCount1 := 0
	answerCount2 := 0
	s1 := newServer(func(w dns.ResponseWriter, r *dns.Msg) {
		if r.Question[0].Name == "example1." {
			msg := dns.Msg{
				Answer: []dns.RR{makeRecordA("example1 3600	IN	A 10.0.0.1")},
			}
			answerCount1++
			msg.SetReply(r)
			w.WriteMsg(&msg)
		}
	}, "example1.")
	s2 := newServer(func(w dns.ResponseWriter, r *dns.Msg) {
		if r.Question[0].Name == "example2." {
			msg := dns.Msg{
				Answer: []dns.RR{makeRecordA("example2. 3600	IN	A 10.0.0.1")},
			}
			answerCount2++
			msg.SetReply(r)
			w.WriteMsg(&msg)
		}
	}, "example2.")
	defer s1.close()
	defer s2.close()

	c1 := createDNSClient(s1.Addr, transport.DNS)
	c2 := createDNSClient(s2.Addr, transport.DNS)
	f := NewFanout()
	f.addClient(c1)
	f.addClient(c2)
	defer f.Close()

	req := new(dns.Msg)
	req.SetQuestion("example1.", dns.TypeA)
	f.ServeDNS(context.TODO(), &test.ResponseWriter{}, req)
	<-time.After(time.Second * 5)
	req = new(dns.Msg)
	req.SetQuestion("example2.", dns.TypeA)
	f.ServeDNS(context.TODO(), &test.ResponseWriter{}, req)

	if answerCount2 != expected || answerCount1 != expected {
		t.Errorf("Expected number of health checks to be %d, got s1: %d, s2: %d", expected, answerCount1, answerCount2)
	}
}

func TestFanout(t *testing.T) {
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
