package fanout

import (
	"fmt"

	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/parse"
	"github.com/coredns/coredns/plugin/pkg/transport"

	"github.com/caddyserver/caddy"
	"github.com/caddyserver/caddy/caddyfile"
)

func init() {
	caddy.RegisterPlugin("fanout", caddy.Plugin{
		ServerType: "dns",
		Action:     setup,
	})
}

func setup(c *caddy.Controller) error {
	f, err := parseFanout(c)
	if err != nil {
		return plugin.Error("fanout", err)
	}
	if f.Len() > maxIPCount {
		return plugin.Error("fanout", fmt.Errorf("more than %d TOs configured: %d", maxIPCount, f.Len()))
	}

	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		f.Next = next
		return f
	})

	c.OnStartup(func() error {
		return f.OnStartup()
	})

	c.OnShutdown(func() error {
		return f.OnShutdown()
	})

	return nil
}

// OnStartup starts a goroutines for all clients.
func (f *Fanout) OnStartup() (err error) {
	for _, p := range f.clients {
		p.start()
	}
	return nil
}

// OnShutdown stops all configured clients.
func (f *Fanout) OnShutdown() error {
	return nil
}

// Close is a synonym for OnShutdown().
func (f *Fanout) Close() {
	checkErr(f.OnShutdown())
}

func parseFanout(c *caddy.Controller) (*Fanout, error) {
	var (
		f   *Fanout
		err error
		i   int
	)
	for c.Next() {
		if i > 0 {
			return nil, plugin.ErrOnce
		}
		i++
		f, err = ParsefanoutStanza(&c.Dispenser)
		if err != nil {
			return nil, err
		}
	}
	return f, nil
}

// ParsefanoutStanza parses one fanout stanza
func ParsefanoutStanza(c *caddyfile.Dispenser) (*Fanout, error) {
	f := NewFanout()

	to := c.RemainingArgs()
	if len(to) == 0 {
		return f, c.ArgErr()
	}

	toHosts, err := parse.HostPortOrFile(to...)
	if err != nil {
		return f, err
	}

	transports := make([]string, len(toHosts))
	for i, host := range toHosts {
		trans, h := parse.Transport(host)
		p := createFanoutClient(h)
		f.clients = append(f.clients, p)
		transports[i] = trans
	}

	for c.NextBlock() {
		return nil, c.Errf("unknown property: %s", c.Val())
	}

	if f.tlsServerName != "" {
		f.tlsConfig.ServerName = f.tlsServerName
	}
	for i := range f.clients {
		// Only set this for clients that need it.
		if transports[i] == transport.TLS {
			f.clients[i].setTLSConfig(f.tlsConfig)
		}
		f.clients[i].setExpire(f.expire)
	}
	return f, nil
}
