package fanout

import (
	"fmt"
	"testing"

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

	for i := 0; i < 100; i++ {
		p := createFanoutClient(s.Addr)
		p.start()

		fn := func() {
			_, connErr := p.Connect(req)
			fmt.Println(connErr)
		}

		go fn()
		go fn()
	}
}

func TestProtocol(t *testing.T) {
	p := createFanoutClient("bad_address")

	req := request.Request{W: &test.ResponseWriter{TCP: true}, Req: new(dns.Msg)}

	go func() {
		_, connErr := p.Connect(req)
		fmt.Println(connErr)
	}()

	proto := <-p.transport.dial
	p.transport.ret <- nil
	if proto != "tcp" {
		t.Errorf("Unexpected protocol in expected tcp, actual %q", proto)
	}
}
