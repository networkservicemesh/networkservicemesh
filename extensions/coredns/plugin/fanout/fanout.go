package fanout

import (
	"context"
	"crypto/tls"
	"errors"
	"time"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/debug"
	clog "github.com/coredns/coredns/plugin/pkg/log"
	"github.com/coredns/coredns/request"

	"github.com/miekg/dns"
)

var log = clog.NewWithPlugin("fanout")

// Fanout represents a plugin instance that can proxy requests to another (DNS) server. It has a list
// of nextUnits each representing one upstream proxy.
type Fanout struct {
	nextUnits  []*DnsServerDefinition
	hcInterval time.Duration

	from    string
	ignored []string

	tlsConfig     *tls.Config
	tlsServerName string
	failLimit     uint32
	expire        time.Duration

	Next plugin.Handler
}

type fanoutResult struct {
	server   *DnsServerDefinition
	response *dns.Msg
	start    time.Time
}

// New returns a new Fanout.
func New() *Fanout {
	f := &Fanout{failLimit: 2, tlsConfig: new(tls.Config), expire: defaultExpire, from: ".", hcInterval: hcInterval}
	return f
}

// SetProxy appends p to the proxy list and starts healthchecking.
func (f *Fanout) SetProxy(p *DnsServerDefinition) {
	f.nextUnits = append(f.nextUnits, p)
	p.start(f.hcInterval)
}

// Len returns the number of configured nextUnits.
func (f *Fanout) Len() int { return len(f.nextUnits) }

// Name implements plugin.Handler.
func (f *Fanout) Name() string { return "fanout" }

// ServeDNS implements plugin.Handler.
func (f *Fanout) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {

	state := request.Request{W: w, Req: r}
	if !f.match(state) {
		return plugin.NextOrFailure(f.Name(), f.Next, ctx, w, r)
	}

	result := make(chan fanoutResult, len(f.nextUnits))

	for i, _ := range f.nextUnits {
		s := request.Request{W: w, Req: r.Copy()}
		unit := f.nextUnits[i]
		go func() {
			start := time.Now()
			for attepmpt := 0; attepmpt < 2; attepmpt++ {
				resp, err := unit.Connect(ctx, s)

				if err == ErrCachedClosed {
					continue
				}

				if err == nil && len(resp.Answer) != 0 {
					log.Infof("got msg:%v", resp)
					result <- fanoutResult{
						server:   unit,
						response: resp,
						start:    start,
					}
					return
				}
			}
		}()
	}
	var first fanoutResult
	select {
	case <-time.After(defaultTimeout):
		return dns.RcodeServerFailure, ErrNoHealthy
	case first = <-result:
		break
	}
	taperr := toDnstap(ctx, first.server.addr, f, state, first.response, first.start)

	// Check if the reply is correct; if not return FormErr.
	if !state.Match(first.response) {
		debug.Hexdumpf(first.response, "Wrong reply for id: %d, %s %d", first.response.Id, state.QName(), state.QType())

		formerr := new(dns.Msg)
		formerr.SetRcode(state.Req, dns.RcodeFormatError)
		w.WriteMsg(formerr)
		return 0, taperr
	}

	w.WriteMsg(first.response)
	return 0, taperr
}

func (f *Fanout) match(state request.Request) bool {
	if !plugin.Name(f.from).Matches(state.Name()) || !f.isAllowedDomain(state.Name()) {
		return false
	}

	return true
}

func (f *Fanout) isAllowedDomain(name string) bool {
	if dns.Name(name) == dns.Name(f.from) {
		return true
	}

	for _, ignore := range f.ignored {
		if plugin.Name(ignore).Matches(name) {
			return false
		}
	}
	return true
}

var (
	// ErrNoHealthy means no healthy nextUnits left.
	ErrNoHealthy = errors.New("no healthy nextUnits")
	// ErrNoForward means no forwarder defined.
	ErrNoForward = errors.New("no forwarder defined")
	// ErrCachedClosed means cached connection was closed by peer.
	ErrCachedClosed = errors.New("cached connection was closed by peer")
)

const defaultTimeout = 5 * time.Second
