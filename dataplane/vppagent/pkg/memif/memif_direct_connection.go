package memif

import (
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/ligato/networkservicemesh/dataplane/pkg/memifproxy"
	"github.com/sirupsen/logrus"
	"os"
	"path"
)

type DirectMemifConnector struct {
	proxyMap map[string]*memifproxy.Proxy
	baseDir  string
}

func NewDirectMemifConnector(baseDir string) *DirectMemifConnector {
	return &DirectMemifConnector{
		proxyMap: make(map[string]*memifproxy.Proxy),
		baseDir:  baseDir,
	}
}

func (d *DirectMemifConnector) ConnectOrDisConnect(crossConnect *crossconnect.CrossConnect, connect bool) (*crossconnect.CrossConnect, error) {
	if connect {
		return d.connect(crossConnect)
	}
	d.disconnect(crossConnect)
	return crossConnect, nil
}

func (d *DirectMemifConnector) connect(crossConnect *crossconnect.CrossConnect) (*crossconnect.CrossConnect, error) {
	logrus.Infof("Direct memif cross connect request: %v", crossConnect)

	if _, exist := d.proxyMap[crossConnect.Id]; exist {
		logrus.Warnf("Proxy for cross connect with id=%s already exists", crossConnect.Id)
		return crossConnect, nil
	}

	src := crossConnect.GetLocalSource().GetMechanism()
	dst := crossConnect.GetLocalDestination().GetMechanism()

	fullyQualifiedDstSocketFilename := path.Join(d.baseDir, dst.GetWorkspace(), dst.GetSocketFilename())
	fullyQualifiedSrcSocketFilename := path.Join(d.baseDir, src.GetWorkspace(), src.GetSocketFilename())

	if err := os.MkdirAll(path.Dir(fullyQualifiedSrcSocketFilename), 0777); err != nil {
		return nil, err
	}
	logrus.Infof("Successfully created directory: %v", path.Dir(fullyQualifiedSrcSocketFilename))

	proxy := memifproxy.NewProxy(fullyQualifiedSrcSocketFilename, fullyQualifiedDstSocketFilename)
	if err := proxy.Start(); err != nil {
		return nil, err
	}

	d.proxyMap[crossConnect.Id] = proxy
	logrus.Infof("Add new proxy for cross connect with id=%s", crossConnect.Id)
	return crossConnect, nil
}

func (d *DirectMemifConnector) disconnect(crossConnect *crossconnect.CrossConnect) {
	proxy, exist := d.proxyMap[crossConnect.Id]
	if !exist {
		logrus.Warnf("Proxy for cross connect with id=%s doesn't exist. Nothing to stop")
		return
	}
	proxy.Stop()
	delete(d.proxyMap, crossConnect.Id)
}
