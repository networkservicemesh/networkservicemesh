package fanout

import (
	"io"
	"sync/atomic"
	"time"

	"github.com/coredns/coredns/request"
	"github.com/pkg/errors"

	"github.com/miekg/dns"
)

func limitTimeout(currentAvg *int64, minValue, maxValue time.Duration) time.Duration {
	rt := time.Duration(atomic.LoadInt64(currentAvg))
	if rt < minValue {
		return minValue
	}
	if rt < maxValue/2 {
		return 2 * rt
	}
	return maxValue
}

func averageTimeout(currentAvg *int64, observedDuration time.Duration, weight int64) {
	dt := time.Duration(atomic.LoadInt64(currentAvg))
	atomic.AddInt64(currentAvg, int64(observedDuration-dt)/weight)
}

func (t *Transport) dialTimeout() time.Duration {
	return limitTimeout(&t.avgDialTime, minDialTimeout, maxDialTimeout)
}

func (t *Transport) updateDialTimeout(newDialTime time.Duration) {
	averageTimeout(&t.avgDialTime, newDialTime, cumulativeAvgWeight)
}

// Dial dials the address configured in Transport, potentially reusing a connection or creating a new one.
func (t *Transport) Dial(protocol string) (*dns.Conn, bool, error) {
	if t.tlsConfig != nil {
		protocol = tcptlc
	}

	t.dial <- protocol
	c := <-t.ret

	if c != nil {
		return c, true, nil
	}

	reqTime := time.Now()
	timeout := t.dialTimeout()
	if protocol == tcptlc {
		conn, err := dns.DialTimeoutWithTLS("tcp", t.addr, t.tlsConfig, timeout)
		t.updateDialTimeout(time.Since(reqTime))
		return conn, false, err
	}
	conn, err := dns.DialTimeout(protocol, t.addr, timeout)
	t.updateDialTimeout(time.Since(reqTime))
	return conn, false, err
}

// Connect selects an upstream, sends the request and waits for a response.
func (p *fanoutClient) Connect(request request.Request) (*dns.Msg, error) {
	proto := "tcp"

	conn, cached, dialErr := p.transport.Dial(proto)
	if dialErr != nil {
		return nil, dialErr
	}

	err := conn.SetWriteDeadline(time.Now().Add(maxTimeout))
	if err != nil {
		log.Error(err)
	}
	if dialErr = conn.WriteMsg(request.Req); dialErr != nil {
		err = conn.Close()
		if err != nil {
			log.Error(err)
		}
		if dialErr == io.EOF && cached {
			return nil, errors.WithStack(errCachedClosed)
		}
		return nil, dialErr
	}

	var ret *dns.Msg
	err = conn.SetReadDeadline(time.Now().Add(readTimeout))
	if err != nil {
		log.Error(err)
	}
	for {
		ret, dialErr = conn.ReadMsg()
		if dialErr != nil {
			err = conn.Close()
			if err != nil {
				log.Error(err)
			}
			if dialErr == io.EOF && cached {
				return nil, errors.WithStack(errCachedClosed)
			}
			return ret, dialErr
		}
		if request.Req.Id == ret.Id {
			break
		}
	}

	p.transport.Yield(conn)

	return ret, nil
}
