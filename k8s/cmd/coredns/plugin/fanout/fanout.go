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

// Fanout represents makeRecordA plugin instance that can do async proxy requests to another (DNS) servers.
type Fanout struct {
	agents []*dnsAgent

	tlsConfig     *tls.Config
	tlsServerName string
	failLimit     int
	expire        time.Duration

	Next plugin.Handler
}

type connectResult struct {
	server   *dnsAgent
	response *dns.Msg
	start    time.Time
}

// NewFanout returns reference to new Fanout plugin instance with default configs.
func NewFanout() *Fanout {
	f := &Fanout{
		failLimit: 2,
		tlsConfig: new(tls.Config),
		expire:    defaultExpire,
	}
	return f
}

// setProxy appends p to the proxy list and starts healthchecking.
func (f *Fanout) setProxy(p *dnsAgent) {
	f.agents = append(f.agents, p)
	p.start(healthClientInterval)
}

// Len returns the number of configured agents.
func (f *Fanout) Len() int {
	return len(f.agents)
}

// Name implements plugin.Handler.
func (f *Fanout) Name() string {
	return "fanout"
}

// ServeDNS implements plugin.Handler.
func (f *Fanout) ServeDNS(ctx context.Context, w dns.ResponseWriter, m *dns.Msg) (int, error) {
	req := request.Request{W: w, Req: m}
	result := make(chan connectResult, len(f.agents))
	for i, _ := range f.agents {
		agent := f.agents[i]
		go func() {
			start := time.Now()
			for attempt := 0; attempt < f.failLimit; attempt++ {
				resp, err := agent.Connect(ctx, req)
				if err == errCachedClosed {
					continue
				}
				if err == nil && len(resp.Answer) != 0 {
					result <- connectResult{
						server:   agent,
						response: resp,
						start:    start,
					}
					return
				}
				if err != nil {
					agent.healthcheck()
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
	taperr := toDnstap(ctx, first.server.addr, f, req, first.response, first.start)
	// Check if the reply is correct; if not return FormErr.
	if !req.Match(first.response) {
		debug.Hexdumpf(first.response, "Wrong reply for id: %d, %s %d", first.response.Id, req.QName(), req.QType())
		formerr := new(dns.Msg)
		formerr.SetRcode(req.Req, dns.RcodeFormatError)
		w.WriteMsg(formerr)
		return 0, taperr
	}

	w.WriteMsg(first.response)
	return 0, taperr
}
