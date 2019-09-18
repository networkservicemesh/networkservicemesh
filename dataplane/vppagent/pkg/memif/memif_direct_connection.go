package memif

import (
	"os"
	"path"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/dataplane/pkg/memifproxy"
)

type DirectMemifConnector struct {
	proxyMap *sync.Map
	baseDir  string
}

func NewDirectMemifConnector(baseDir string) *DirectMemifConnector {
	return &DirectMemifConnector{
		proxyMap: &sync.Map{},
		baseDir:  baseDir,
	}
}

// ConnectOrDisconnect handler for request() or close() connections
func (d *DirectMemifConnector) ConnectOrDisconnect(crossConnect *crossconnect.CrossConnect, connect bool) (*crossconnect.CrossConnect, error) {
	if connect {
		return d.connect(crossConnect)
	}
	d.disconnect(crossConnect)
	return crossConnect, nil
}

func (d *DirectMemifConnector) connect(crossConnect *crossconnect.CrossConnect) (*crossconnect.CrossConnect, error) {
	logrus.Infof("Direct memif cross connect request: %v", crossConnect)

	src := crossConnect.GetLocalSource().GetMechanism()
	dst := crossConnect.GetLocalDestination().GetMechanism()

	fullyQualifiedDstSocketFilename := path.Join(d.baseDir, dst.GetWorkspace(), dst.GetSocketFilename())
	fullyQualifiedSrcSocketFilename := path.Join(d.baseDir, src.GetWorkspace(), src.GetSocketFilename())

	proxy, err := memifproxy.NewProxy(fullyQualifiedSrcSocketFilename, fullyQualifiedDstSocketFilename)

	if err != nil {
		return nil, err
	}

	_, exist := d.proxyMap.LoadOrStore(crossConnect.Id, proxy)

	if exist {
		logrus.Warnf("Proxy for cross connect with id=%s already exists", crossConnect.Id)
		return crossConnect, nil
	}

	if err := os.MkdirAll(path.Dir(fullyQualifiedSrcSocketFilename), 0777); err != nil {
		return nil, err
	}
	logrus.Infof("Successfully created directory: %v", path.Dir(fullyQualifiedSrcSocketFilename))

	if err := proxy.Start(); err != nil {
		return nil, err
	}

	logrus.Infof("Add new proxy for cross connect with id=%s", crossConnect.Id)
	return crossConnect, nil
}

func (d *DirectMemifConnector) disconnect(crossConnect *crossconnect.CrossConnect) {
	value, exist := d.proxyMap.Load(crossConnect.GetId())
	if !exist {
		logrus.Warnf("Proxy for cross connect with id=%s doesn't exist. Nothing to stop", crossConnect.GetId())
		return
	}

	proxy := value.(*memifproxy.Proxy)
	proxy.Stop()

	d.proxyMap.Delete(crossConnect.Id)
}
