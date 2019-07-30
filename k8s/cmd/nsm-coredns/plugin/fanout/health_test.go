package fanout

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coredns/coredns/plugin/pkg/transport"
	"github.com/coredns/coredns/plugin/test"

	"github.com/miekg/dns"
)

func TestHealth(t *testing.T) {
	const expected = 0
	i := uint32(0)
	s := newServer(func(w dns.ResponseWriter, r *dns.Msg) {
		if r.Question[0].Name == "." {
			atomic.AddUint32(&i, 1)
		}
		ret := new(dns.Msg)
		ret.SetReply(r)
		w.WriteMsg(ret)
	}, ".")
	defer s.close()
	<-time.After(time.Second)
	p := createDNSClient(s.Addr, transport.DNS)
	f := NewFanout()
	f.addClient(p)
	defer f.Close()

	req := new(dns.Msg)
	req.SetQuestion("example.org.", dns.TypeA)

	f.ServeDNS(context.TODO(), &test.ResponseWriter{}, req)

	<-time.After(time.Second)
	i1 := atomic.LoadUint32(&i)
	if i1 != expected {
		t.Errorf("Expected number of health checks to be %d, got %d", expected, i1)
	}
}

func TestHealthFailTwice(t *testing.T) {
	const expected = 2
	i := uint32(0)
	q := uint32(0)
	s := newServer(func(w dns.ResponseWriter, r *dns.Msg) {
		if r.Question[0].Name == "." {
			atomic.AddUint32(&i, 1)
			i1 := atomic.LoadUint32(&i)
			// Timeout health until we get the second one
			if i1 < 2 {
				return
			}
			ret := new(dns.Msg)
			ret.SetReply(r)

			w.WriteMsg(ret)
			return
		}
		if atomic.LoadUint32(&q) == 0 { //drop only first query
			atomic.AddUint32(&q, 1)
			return
		}
		ret := new(dns.Msg)
		ret.SetReply(r)
		w.WriteMsg(ret)
	}, ".")
	defer s.close()

	p := createDNSClient(s.Addr, transport.DNS)
	f := NewFanout()
	f.addClient(p)
	defer f.Close()

	req := new(dns.Msg)
	req.SetQuestion("example.org", dns.TypeA)

	f.ServeDNS(context.TODO(), &test.ResponseWriter{}, req)

	<-time.After(time.Second * 5)
	i1 := atomic.LoadUint32(&i)
	if i1 != expected {
		t.Errorf("Expected number of health checks to be %d, got %d", expected, i1)
	}
}

func TestHealthNoMaxFails(t *testing.T) {
	const expected = 0
	i := uint32(0)
	s := newServer(func(w dns.ResponseWriter, r *dns.Msg) {
		if r.Question[0].Name == "." {
			atomic.AddUint32(&i, 1)
			ret := new(dns.Msg)
			ret.SetReply(r)
			w.WriteMsg(ret)
		}
	}, ".")
	defer s.close()

	p := createDNSClient(s.Addr, transport.DNS)
	f := NewFanout()
	f.maxFailCount = 0
	f.addClient(p)
	defer f.Close()

	req := new(dns.Msg)
	req.SetQuestion("example.org.", dns.TypeA)

	f.ServeDNS(context.TODO(), &test.ResponseWriter{}, req)

	time.Sleep(1 * time.Second)
	i1 := atomic.LoadUint32(&i)
	if i1 != expected {
		t.Errorf("Expected number of health checks to be %d, got %d", expected, i1)
	}
}
