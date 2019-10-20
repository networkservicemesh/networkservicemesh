package fanout

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/coredns/coredns/plugin/pkg/dnstest"
	"github.com/coredns/coredns/plugin/test"
	"github.com/coredns/coredns/request"

	"github.com/miekg/dns"
)

func TestDnsClientClose(t *testing.T) {
	s := dnstest.NewServer(func(w dns.ResponseWriter, r *dns.Msg) {
		ret := new(dns.Msg)
		ret.SetReply(r)
		checkErr(w.WriteMsg(ret))
	})
	defer s.Close()

	msg := new(dns.Msg)
	msg.SetQuestion("example.org.", dns.TypeA)
	req := request.Request{W: &test.ResponseWriter{}, Req: msg}

	wg := sync.WaitGroup{}
	for i := 0; i < 100; i++ {
		p := createFanoutClient(s.Addr)
		p.start()
		wg.Add(2)

		fn := func() {
			_, connErr := p.Connect(req)
			assert.NoError(t, connErr)
			wg.Done()
		}

		go fn()
		go fn()
	}
	wg.Wait()
}

func TestProtocol(t *testing.T) {
	p := createFanoutClient("bad_address")

	req := request.Request{W: &test.ResponseWriter{TCP: true}, Req: new(dns.Msg)}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		_, connErr := p.Connect(req)
		assert.NoError(t, connErr)
		wg.Done()
	}()
	wg.Wait()

	proto := <-p.transport.dial
	p.transport.ret <- nil
	if proto != "tcp" {
		t.Errorf("Unexpected protocol in expected tcp, actual %q", proto)
	}
}
