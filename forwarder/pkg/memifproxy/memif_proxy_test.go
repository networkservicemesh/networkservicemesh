package memifproxy

import (
	"net"
	"os"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func TestClosingOpeningMemifProxy(t *testing.T) {
	g := NewWithT(t)
	proxy, err := newCustomProxy("source.sock", "target.sock", "unix", nil)
	g.Expect(err).Should(BeNil())
	for i := 0; i < 10; i++ {
		err = proxy.Start()
		g.Expect(err).To(BeNil())
		err = proxy.Stop()
		g.Expect(err).To(BeNil())
	}
}

func TestTransferBetweenMemifProxies(t *testing.T) {
	g := NewWithT(t)
	for i := 0; i < 10; i++ {
		proxy1, err := newCustomProxy("source.sock", "target.sock", "unix", nil)
		g.Expect(err).Should(BeNil())
		proxy2, err := newCustomProxy("target.sock", "source.sock", "unix", nil)
		g.Expect(err).Should(BeNil())
		err = proxy1.Start()
		g.Expect(err).To(BeNil())
		err = proxy2.Start()
		g.Expect(err).To(BeNil())
		err = connectAndSendMsg("source.sock")
		g.Expect(err).To(BeNil())
		err = connectAndSendMsg("target.sock")
		g.Expect(err).To(BeNil())
		err = proxy1.Stop()
		g.Expect(err).To(BeNil())
		err = proxy2.Stop()
		g.Expect(err).To(BeNil())
	}
}

func TestProxyListenerCalled(t *testing.T) {
	proxyStopped := false
	g := NewWithT(t)
	proxy, err := newCustomProxy("source.sock", "target.sock", "unix", StopListenerAdapter(func() {
		proxyStopped = true
	}))
	g.Expect(err).Should(BeNil())
	err = proxy.Start()
	g.Expect(err).To(BeNil())
	err = proxy.Stop()
	g.Expect(err).To(BeNil())
	for t := time.Now(); time.Since(t) < time.Second; {
		if proxyStopped {
			break
		}
	}
	g.Expect(proxyStopped).Should(BeTrue())
}

func TestProxyListenerCalledOnDestroySocketFile(t *testing.T) {
	proxyStopped := false
	g := NewWithT(t)
	proxy, err := newCustomProxy("source.sock", "target.sock", "unix", StopListenerAdapter(func() {
		proxyStopped = true
	}))
	g.Expect(err).Should(BeNil())
	err = proxy.Start()
	g.Expect(err).To(BeNil())
	err = connectAndSendMsg("source.sock")
	g.Expect(err).To(BeNil())
	err = os.Remove("source.sock")
	g.Expect(err).To(BeNil())
	for t := time.Now(); time.Since(t) < time.Second; {
		if proxyStopped {
			break
		}
	}
	g.Expect(proxyStopped).Should(BeTrue())
}

func TestStartProxyIfSocketFileIsExist(t *testing.T) {
	g := NewWithT(t)
	_, err := os.Create("source.sock")
	g.Expect(err).Should(BeNil())
	proxy, err := newCustomProxy("source.sock", "target.sock", "unix", nil)
	g.Expect(err).Should(BeNil())
	err = proxy.Start()
	g.Expect(err).To(BeNil())
	err = proxy.Stop()
	g.Expect(err).To(BeNil())
}

func connectAndSendMsg(sock string) error {
	addr, err := net.ResolveUnixAddr("unix", sock)
	if err != nil {
		return err
	}

	var conn *net.UnixConn
	conn, err = net.DialUnix("unix", nil, addr)
	if err != nil {
		return err
	}

	_, err = conn.Write([]byte("secret"))
	if err != nil {
		return err
	}

	err = conn.Close()
	if err != nil {
		logrus.Error(err.Error())
		return err
	}
	return nil
}
