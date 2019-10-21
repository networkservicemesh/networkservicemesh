package fanout

import (
	"context"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/caddyserver/caddy"
	"github.com/coredns/coredns/plugin/pkg/dnstest"
	"github.com/coredns/coredns/plugin/test"
	"github.com/miekg/dns"
)

type cachedDNSWriter struct {
	answers []*dns.Msg
	mutex   sync.Mutex
	*test.ResponseWriter
}

func (w *cachedDNSWriter) WriteMsg(m *dns.Msg) error {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	w.answers = append(w.answers, m)
	return w.ResponseWriter.WriteMsg(m)
}

type server struct {
	Addr  string
	inner *dns.Server
}

func (s *server) close() {
	if err := s.inner.Shutdown(); err != nil {
		logrus.Error("shutdown failed", err)
	}
}

func newServer(f dns.HandlerFunc) *server {
	ch := make(chan bool)
	s := &dns.Server{}
	s.Handler = f

	for i := 0; i < 10; i++ {
		s.Listener, _ = net.Listen("tcp", ":0")
		if s.Listener != nil {
			break
		}
	}
	if s.Listener == nil {
		panic("failed to create new client")
	}

	s.NotifyStartedFunc = func() { close(ch) }
	go func() {
		if serveErr := s.ActivateAndServe(); serveErr != nil {
			logrus.Error(serveErr)
		}
	}()

	<-ch
	return &server{inner: s, Addr: s.Listener.Addr().String()}
}

func makeRecordA(rr string) *dns.A {
	r, _ := dns.NewRR(rr)
	return r.(*dns.A)
}

func TestFanoutCanReturnUnsuccessRespnse(t *testing.T) {
	s := newServer(func(w dns.ResponseWriter, r *dns.Msg) {
		msg := testNxdomainMsg()
		msg.SetRcode(r, msg.Rcode)
		err := w.WriteMsg(msg)
		assert.NoError(t, err)
	})
	f := NewFanout()
	c := createFanoutClient(s.Addr)
	f.addClient(c)
	defer f.Close()
	req := new(dns.Msg)
	req.SetQuestion("example1.", dns.TypeA)
	writer := &cachedDNSWriter{ResponseWriter: new(test.ResponseWriter)}
	_, err := f.ServeDNS(context.TODO(), writer, req)
	assert.NoError(t, err)
	if len(writer.answers) != 1 {
		fmt.Println(len(writer.answers))
		t.FailNow()
	}
	if writer.answers[0].MsgHdr.Rcode != dns.RcodeNameError {
		t.Error("fanout plugin returns first negative answer if other answers on request are negative")
	}
}
func TestFanoutTwoServersNotSuccessResponse(t *testing.T) {
	rcode := 1
	s1 := newServer(func(w dns.ResponseWriter, r *dns.Msg) {
		if r.Question[0].Name == "example1." {
			msg := testNxdomainMsg()
			msg.SetRcode(r, rcode)
			rcode++
			rcode %= dns.RcodeNotZone
			err := w.WriteMsg(msg)
			assert.NoError(t, err)
			//let another server answer
			<-time.After(time.Millisecond * 100)
		}
	})
	s2 := newServer(func(w dns.ResponseWriter, r *dns.Msg) {
		if r.Question[0].Name == "example1." {
			msg := dns.Msg{
				Answer: []dns.RR{makeRecordA("example1. 3600	IN	A 10.0.0.1")},
			}
			msg.SetReply(r)
			err := w.WriteMsg(&msg)
			assert.NoError(t, err)
			//let another server answer
			<-time.After(time.Millisecond * 100)
		}
	})
	defer s1.close()
	defer s2.close()
	c1 := createFanoutClient(s1.Addr)
	c2 := createFanoutClient(s2.Addr)
	f := NewFanout()
	f.addClient(c1)
	f.addClient(c2)
	defer f.Close()
	req := new(dns.Msg)
	writer := &cachedDNSWriter{ResponseWriter: new(test.ResponseWriter)}
	for i := 0; i < 10; i++ {
		req.SetQuestion("example1.", dns.TypeA)
		_, err := f.ServeDNS(context.TODO(), writer, req)
		assert.NoError(t, err)
	}
	for _, m := range writer.answers {
		if m.MsgHdr.Rcode != dns.RcodeSuccess {
			t.Error("fanout should return only positive answers")
		}
	}
}

func TestFanoutTwoServers(t *testing.T) {
	const expected = 1
	answerCount1 := 0
	answerCount2 := 0
	s1 := newServer(func(w dns.ResponseWriter, r *dns.Msg) {
		if r.Question[0].Name == "example1." {
			msg := dns.Msg{
				Answer: []dns.RR{makeRecordA("example1 3600	IN	A 10.0.0.1")},
			}
			answerCount1++
			msg.SetReply(r)
			err := w.WriteMsg(&msg)
			assert.NoError(t, err)
		}
	})
	s2 := newServer(func(w dns.ResponseWriter, r *dns.Msg) {
		if r.Question[0].Name == "example2." {
			msg := dns.Msg{
				Answer: []dns.RR{makeRecordA("example2. 3600	IN	A 10.0.0.1")},
			}
			answerCount2++
			msg.SetReply(r)
			err := w.WriteMsg(&msg)
			assert.NoError(t, err)
		}
	})
	defer s1.close()
	defer s2.close()

	c1 := createFanoutClient(s1.Addr)
	c2 := createFanoutClient(s2.Addr)
	f := NewFanout()
	f.addClient(c1)
	f.addClient(c2)
	defer f.Close()

	req := new(dns.Msg)
	req.SetQuestion("example1.", dns.TypeA)
	_, err := f.ServeDNS(context.TODO(), &test.ResponseWriter{}, req)
	assert.NoError(t, err)
	<-time.After(time.Second)
	req = new(dns.Msg)
	req.SetQuestion("example2.", dns.TypeA)
	_, err = f.ServeDNS(context.TODO(), &test.ResponseWriter{}, req)
	assert.NoError(t, err)
	<-time.After(time.Second)
	if answerCount2 != expected || answerCount1 != expected {
		t.Errorf("Expected number of health checks to be %d, got s1: %d, s2: %d", expected, answerCount1, answerCount2)
	}
}

func TestFanout(t *testing.T) {
	s := newServer(func(w dns.ResponseWriter, r *dns.Msg) {
		ret := new(dns.Msg)
		ret.SetReply(r)
		ret.Answer = append(ret.Answer, test.A("example.org. IN A 127.0.0.1"))
		err := w.WriteMsg(ret)
		assert.NoError(t, err)
	})
	defer s.close()
	c := caddy.NewTestController("dns", "fanout "+s.Addr)
	f, err := parseFanout(c)
	if err != nil {
		t.Errorf("Failed to create fanout: %s", err)
	}
	err = f.OnStartup()
	assert.NoError(t, err)
	defer func() {
		err = f.OnShutdown()
		assert.NoError(t, err)
	}()

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
func testNxdomainMsg() *dns.Msg {
	return &dns.Msg{MsgHdr: dns.MsgHdr{Rcode: dns.RcodeNameError},
		Question: []dns.Question{{Name: "wwww.example1.", Qclass: dns.ClassINET, Qtype: dns.TypeTXT}},
		Ns: []dns.RR{test.SOA("example1.	1800	IN	SOA	example1.net. example1.com 1461471181 14400 3600 604800 14400")},
	}
}
