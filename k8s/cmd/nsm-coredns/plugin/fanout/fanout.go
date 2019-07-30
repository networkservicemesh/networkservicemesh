package fanout

import (
	"context"
	"crypto/tls"
	"time"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/debug"
	clog "github.com/coredns/coredns/plugin/pkg/log"
	"github.com/coredns/coredns/request"

	"github.com/miekg/dns"
)

var log = clog.NewWithPlugin("fanout")

// Fanout represents a plugin instance that can do async proxy requests to another (DNS) servers.
type Fanout struct {
	clients []*dnsClient

	tlsConfig     *tls.Config
	tlsServerName string
	maxFailCount  int
	expire        time.Duration

	Next plugin.Handler
}

type connectResult struct {
	server   *dnsClient
	response *dns.Msg
	start    time.Time
}

// NewFanout returns reference to new Fanout plugin instance with default configs.
func NewFanout() *Fanout {
	f := &Fanout{
		maxFailCount: 2,
		tlsConfig:    new(tls.Config),
		expire:       defaultExpire,
	}
	return f
}

// addClient appends p to the proxy list and starts healthchecking.
func (f *Fanout) addClient(p *dnsClient) {
	f.clients = append(f.clients, p)
	p.start(healthClientInterval)
}

// Len returns the number of configured clients.
func (f *Fanout) Len() int {
	return len(f.clients)
}

// Name implements plugin.Handler.
func (f *Fanout) Name() string {
	return "fanout"
}

// ServeDNS implements plugin.Handler.
func (f *Fanout) ServeDNS(ctx context.Context, w dns.ResponseWriter, m *dns.Msg) (int, error) {
	req := request.Request{W: w, Req: m}
	result := make(chan connectResult, len(f.clients))
	for i := 0; i < len(f.clients); i++ {
		client := f.clients[i]
		go func() {
			start := time.Now()
			for attempt := 0; attempt < f.maxFailCount; attempt++ {
				resp, err := client.Connect(req)
				if err == errCachedClosed {
					continue
				}
				if err == nil && len(resp.Answer) != 0 {
					result <- connectResult{
						server:   client,
						response: resp,
						start:    start,
					}
					return
				}
				if err != nil && attempt+1 < f.maxFailCount {
					client.healthCheck()
				}
			}
		}()
	}
	var first connectResult
	select {
	case <-time.After(defaultTimeout):
		return dns.RcodeServerFailure, errNoHealthy
	case first = <-result:
		break
	}
	taperr := toDnstap(ctx, first.server.addr, req, first.response, first.start)
	if !req.Match(first.response) {
		debug.Hexdumpf(first.response, "Wrong reply for id: %d, %s %d", first.response.Id, req.QName(), req.QType())
		formerr := new(dns.Msg)
		formerr.SetRcode(req.Req, dns.RcodeFormatError)
		checkErr(w.WriteMsg(formerr))
		return 0, taperr
	}
	checkErr(w.WriteMsg(first.response))
	return 0, taperr
}
