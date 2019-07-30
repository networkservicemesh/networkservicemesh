package fanout

import (
	"crypto/tls"
	"net"
	"sort"
	"time"

	"github.com/miekg/dns"
)

type persistConn struct {
	c    *dns.Conn
	used time.Time
}

// Transport hold the persistent cache.
type Transport struct {
	avgDialTime int64
	connections map[string][]*persistConn
	expire      time.Duration
	addr        string
	tlsConfig   *tls.Config

	dial  chan string
	yield chan *dns.Conn
	ret   chan *dns.Conn
	stop  chan bool
}

func newTransport(addr string) *Transport {
	t := &Transport{
		avgDialTime: int64(maxDialTimeout / 2),
		connections: make(map[string][]*persistConn),
		expire:      defaultExpire,
		addr:        addr,
		dial:        make(chan string),
		yield:       make(chan *dns.Conn),
		ret:         make(chan *dns.Conn),
		stop:        make(chan bool),
	}
	return t
}

func (t *Transport) len() int {
	l := 0
	for _, conns := range t.connections {
		l += len(conns)
	}
	return l
}

func (t *Transport) manageConnections() {
	ticker := time.NewTicker(t.expire)
Wait:
	for {
		select {
		case proto := <-t.dial:
			// take the last used conn - complexity O(1)
			if stack := t.connections[proto]; len(stack) > 0 {
				pc := stack[len(stack)-1]
				if time.Since(pc.used) < t.expire {
					// Found one, remove from pool and return this conn.
					t.connections[proto] = stack[:len(stack)-1]
					t.ret <- pc.c
					continue Wait
				}
				// clear entire cache if the last conn is expired
				t.connections[proto] = nil
				// now, the connections being passed to closeConnections() are not reachable from
				// Transport methods anymore. So, it's safe to close them in a separate goroutine
				go closeConnections(stack)
			}

			t.ret <- nil

		case conn := <-t.yield:

			// no proto here, infer from config and conn
			if _, ok := conn.Conn.(*net.UDPConn); ok {
				t.connections["udp"] = append(t.connections["udp"], &persistConn{conn, time.Now()})
				continue Wait
			}

			if t.tlsConfig == nil {
				t.connections["tcp"] = append(t.connections["tcp"], &persistConn{conn, time.Now()})
				continue Wait
			}

			t.connections[tcptlc] = append(t.connections[tcptlc], &persistConn{conn, time.Now()})

		case <-ticker.C:
			t.cleanup(false)

		case <-t.stop:
			t.cleanup(true)
			close(t.ret)
			return
		}
	}
}

func closeConnections(conns []*persistConn) {
	for _, pc := range conns {
		checkErr(pc.c.Close())
	}
}

func (t *Transport) cleanup(all bool) {
	staleTime := time.Now().Add(-t.expire)
	for proto, stack := range t.connections {
		currStack := stack
		if len(stack) == 0 {
			continue
		}
		if all {
			t.connections[proto] = nil
			go closeConnections(stack)
			continue
		}
		if stack[0].used.After(staleTime) {
			continue
		}

		good := sort.Search(len(stack), func(i int) bool {
			return currStack[i].used.After(staleTime)
		})
		t.connections[proto] = stack[good:]
		go closeConnections(stack[:good])
	}
}

// Yield return the connection to Transport for reuse.
func (t *Transport) Yield(c *dns.Conn) { t.yield <- c }

// Start starts the Transport's connection manager.
func (t *Transport) Start() { go t.manageConnections() }

// Stop stops the Transport's connection manager.
func (t *Transport) Stop() {
	close(t.stop)
}

func (t *Transport) setExpire(expire time.Duration) {
	t.expire = expire
}

func (t *Transport) setTLSConfig(cfg *tls.Config) {
	t.tlsConfig = cfg
}
