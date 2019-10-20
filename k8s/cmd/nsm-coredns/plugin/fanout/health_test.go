package fanout

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"

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
		err := w.WriteMsg(ret)
		assert.NoError(t, err)
	})
	defer s.close()
	p := createFanoutClient(s.Addr)
	f := NewFanout()
	f.addClient(p)
	defer f.Close()

	req := new(dns.Msg)
	req.SetQuestion("example.org.", dns.TypeA)

	_, err := f.ServeDNS(context.TODO(), &test.ResponseWriter{}, req)
	assert.NoError(t, err)

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
			if i1 < 2 {
				return
			}
			ret := new(dns.Msg)
			ret.SetReply(r)

			err := w.WriteMsg(ret)
			assert.NoError(t, err)
			return
		}
		if atomic.LoadUint32(&q) == 0 { //drop only first query
			atomic.AddUint32(&q, 1)
			return
		}
		ret := new(dns.Msg)
		ret.SetReply(r)
		err := w.WriteMsg(ret)
		assert.NoError(t, err)
	})
	defer s.close()

	p := createFanoutClient(s.Addr)
	f := NewFanout()
	f.addClient(p)
	defer f.Close()

	req := new(dns.Msg)
	req.SetQuestion("example.org", dns.TypeA)
	_, err := f.ServeDNS(context.TODO(), &test.ResponseWriter{}, req)
	assert.NoError(t, err)
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
			err := w.WriteMsg(ret)
			assert.NoError(t, err)
		}
	})
	defer s.close()

	p := createFanoutClient(s.Addr)
	f := NewFanout()
	f.maxFailCount = 0
	f.addClient(p)
	defer f.Close()

	req := new(dns.Msg)
	req.SetQuestion("example.org.", dns.TypeA)

	_, err := f.ServeDNS(context.TODO(), &test.ResponseWriter{}, req)
	assert.NoError(t, err)
	i1 := atomic.LoadUint32(&i)
	if i1 != expected {
		t.Errorf("Expected number of health checks to be %d, got %d", expected, i1)
	}
}
