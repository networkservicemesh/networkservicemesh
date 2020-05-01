package memifproxy

import (
	"io/ioutil"
	"net"
	"os"
	"path"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func TestClosingOpeningMemifProxy(t *testing.T) {
	g := NewWithT(t)
	tmpFolder, cleanup := genResourceFolder(t.Name())
	defer cleanup()
	sourceSocket, targetSocket := path.Join(tmpFolder, "source.sock"), path.Join(tmpFolder, "target.sock")
	proxy, err := newCustomProxy(sourceSocket, targetSocket, "unix", nil)
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
	tmpFolder, cleanup := genResourceFolder(t.Name())
	defer cleanup()
	sourceSocket, targetSocket := path.Join(tmpFolder, "source.sock"), path.Join(tmpFolder, "target.sock")
	for i := 0; i < 10; i++ {
		proxy1, err := newCustomProxy(sourceSocket, targetSocket, "unix", nil)
		g.Expect(err).Should(BeNil())
		proxy2, err := newCustomProxy(targetSocket, sourceSocket, "unix", nil)
		g.Expect(err).Should(BeNil())
		err = proxy1.Start()
		g.Expect(err).To(BeNil())
		err = proxy2.Start()
		g.Expect(err).To(BeNil())
		err = connectAndSendMsg(sourceSocket)
		g.Expect(err).To(BeNil())
		err = connectAndSendMsg(targetSocket)
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
	tmpFolder, cleanup := genResourceFolder(t.Name())
	defer cleanup()
	sourceSocket, targetSocket := path.Join(tmpFolder, "source.sock"), path.Join(tmpFolder, "target.sock")
	proxy, err := newCustomProxy(sourceSocket, targetSocket, "unix", StopListenerAdapter(func() {
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
	tmpFolder, cleanup := genResourceFolder(t.Name())
	defer cleanup()
	sourceSocket, targetSocket := path.Join(tmpFolder, "source.sock"), path.Join(tmpFolder, "target.sock")
	proxy, err := newCustomProxy(sourceSocket, targetSocket, "unix", StopListenerAdapter(func() {
		proxyStopped = true
	}))
	g.Expect(err).Should(BeNil())
	err = proxy.Start()
	g.Expect(err).To(BeNil())
	err = connectAndSendMsg(sourceSocket)
	g.Expect(err).To(BeNil())
	err = os.Remove(sourceSocket)
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
	tmpFolder, cleanup := genResourceFolder(t.Name())
	defer cleanup()
	sourceSocket, targetSocket := path.Join(tmpFolder, "source.sock"), path.Join(tmpFolder, "target.sock")
	_, err := os.Create(sourceSocket)
	g.Expect(err).Should(BeNil())
	proxy, err := newCustomProxy(sourceSocket, targetSocket, "unix", nil)
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

func genResourceFolder(name string) (string, func()) {
	dir, _ := ioutil.TempDir("", name)
	return dir, func() {
		_ = os.Remove(dir)
	}
}
