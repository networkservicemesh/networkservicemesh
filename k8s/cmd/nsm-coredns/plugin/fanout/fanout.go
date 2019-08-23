package fanout

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/debug"
	clog "github.com/coredns/coredns/plugin/pkg/log"
	"github.com/coredns/coredns/request"

	"github.com/miekg/dns"
)

var log = clog.NewWithPlugin("fanout")

// Fanout represents a plugin instance that can do async proxy requests to another (DNS) servers.
type Fanout struct {
	clients []*fanoutClient

	tlsConfig     *tls.Config
	tlsServerName string
	maxFailCount  int
	expire        time.Duration

	Next plugin.Handler
}

type connectResult struct {
	client   *fanoutClient
	response *dns.Msg
	start    time.Time
	err      error
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

func (f *Fanout) addClient(p *fanoutClient) {
	f.clients = append(f.clients, p)
	p.start()
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
	timeoutContext, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	clientCount := len(f.clients)
	result := make(chan connectResult, clientCount)
	for i := 0; i < clientCount; i++ {
		client := f.clients[i]
		go connect(timeoutContext, client, req, result, f.maxFailCount)
	}
	var first *connectResult
	for first == nil {
		select {
		case <-timeoutContext.Done():
			return dns.RcodeServerFailure, errContextDone
		case conn := <-result:
			if conn.err != nil {
				clientCount--
				if clientCount == 0 {
					return dns.RcodeServerFailure, errNoHealthy
				}
				break
			}
			first = &conn
		}
	}

	taperr := toDnstap(ctx, first.client.addr, req, first.response, first.start)
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

func connect(ctx context.Context, client *fanoutClient, req request.Request, result chan<- connectResult, maxFailCount int) {
	start := time.Now()
	var errs error
	for i := 0; i < maxFailCount+1; i++ {
		resp, err := client.Connect(req)
		if ctx.Err() != nil {
			return
		}
		fmt.Printf("fanout debug: resp %v", resp)
		if err == nil && len(resp.Answer) > 0 {
			result <- connectResult{
				client:   client,
				response: resp,
				start:    start,
			}
			return
		}
		if errs == nil {
			errs = err
		} else {
			errs = errors.Wrap(errs, err.Error())
		}
		if i < maxFailCount {
			if err = client.healthCheck(); err != nil {
				errs = errors.Wrap(errs, err.Error())
				break
			}
		}
	}
	result <- connectResult{
		client: client,
		err:    errs,
		start:  start,
	}
}
