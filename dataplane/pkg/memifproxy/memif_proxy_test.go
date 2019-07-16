package memifproxy

import (
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"net"
	"os"
	"testing"
)

func TestClosingOpeningMemifProxy(t *testing.T) {
	RegisterTestingT(t)
	proxy, err := newCustomProxy("source.sock", "target.sock", "unix")
	Expect(err).Should(BeNil())
	for i := 0; i < 10; i++ {
		startProxy(proxy)
		stopProxy(proxy)
	}
}

func TestTransferBetweenMemifProxies(t *testing.T) {
	RegisterTestingT(t)
	for i := 0; i < 10; i++ {
		proxy1, err := newCustomProxy("source.sock", "target.sock", "unix")
		Expect(err).Should(BeNil())
		proxy2, err := newCustomProxy("target.sock", "source.sock", "unix")
		Expect(err).Should(BeNil())
		startProxy(proxy1)
		startProxy(proxy2)
		connectAndSendMsg("source.sock")
		connectAndSendMsg("target.sock")
		stopProxy(proxy1)
		stopProxy(proxy2)
	}
}

func TestStartProxyIfSocketFileIsExist(t *testing.T) {
	RegisterTestingT(t)
	_, err := os.Create("source.sock")
	Expect(err).Should(BeNil())
	proxy, err := newCustomProxy("source.sock", "target.sock", "unix")
	Expect(err).Should(BeNil())
	startProxy(proxy)
	stopProxy(proxy)
}

func connectAndSendMsg(sock string) {
	addr, err := net.ResolveUnixAddr("unix", sock)
	Expect(err).To(BeNil())
	var conn *net.UnixConn
	conn, err = net.DialUnix("unix", nil, addr)
	Expect(err).To(BeNil())
	_, err = conn.Write([]byte("secret"))
	Expect(err).To(BeNil())
	err = conn.Close()
	if err != nil {
		logrus.Error(err.Error())
	}
	Expect(err).To(BeNil())
}

func startProxy(proxy *Proxy) {
	err := proxy.Start()
	Expect(err).To(BeNil())
}

func stopProxy(proxy *Proxy) {
	err := proxy.Stop()
	Expect(err).To(BeNil())
}
