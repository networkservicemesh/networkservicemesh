package memifproxy

import (
	"net"
	"os"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func TestClosingOpeningMemifProxy(t *testing.T) {
	g := NewWithT(t)
	proxy, err := newCustomProxy("source.sock", "target.sock", "unix")
	g.Expect(err).Should(BeNil())
	for i := 0; i < 10; i++ {
		err = startProxy(proxy)
		g.Expect(err).To(BeNil())
		err = stopProxy(proxy)
		g.Expect(err).To(BeNil())
	}
}

func TestTransferBetweenMemifProxies(t *testing.T) {
	g := NewWithT(t)
	for i := 0; i < 10; i++ {
		proxy1, err := newCustomProxy("source.sock", "target.sock", "unix")
		g.Expect(err).Should(BeNil())
		proxy2, err := newCustomProxy("target.sock", "source.sock", "unix")
		g.Expect(err).Should(BeNil())
		err = startProxy(proxy1)
		g.Expect(err).To(BeNil())
		err = startProxy(proxy2)
		g.Expect(err).To(BeNil())
		err = connectAndSendMsg("source.sock")
		g.Expect(err).To(BeNil())
		err = connectAndSendMsg("target.sock")
		g.Expect(err).To(BeNil())
		err = stopProxy(proxy1)
		g.Expect(err).To(BeNil())
		err = stopProxy(proxy2)
		g.Expect(err).To(BeNil())
	}
}

func TestStartProxyIfSocketFileIsExist(t *testing.T) {
	g := NewWithT(t)
	_, err := os.Create("source.sock")
	g.Expect(err).Should(BeNil())
	proxy, err := newCustomProxy("source.sock", "target.sock", "unix")
	g.Expect(err).Should(BeNil())
	err = startProxy(proxy)
	g.Expect(err).To(BeNil())
	err = stopProxy(proxy)
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

func startProxy(proxy *Proxy) error {
	return proxy.Start()
}

func stopProxy(proxy *Proxy) error {
	return proxy.Stop()
}
