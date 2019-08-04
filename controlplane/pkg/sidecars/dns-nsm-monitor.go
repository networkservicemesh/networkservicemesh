package sidecars

import (
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/sdk/common"
)

const (
	//DefaultPathToCorefile contains default path to Corefile
	DefaultPathToCorefile = "/etc/coredns/Corefile"
	//DefaultReloadCorefileTime means time to reload Corefile
	DefaultReloadCorefileTime = time.Second * 5
	defaultK8sDNSServer       = "10.96.0.10"

	notScheduled = int32(0)
	running      = int32(1)
)

//NsmDNSMonitorHandler implements NSMMonitorHandler interface for handling dnsConfigs
type NsmDNSMonitorHandler struct {
	EmptyNSMMonitorHandler
	updateCorefileState int32
	pathToCorefile      string
	dnsConfigManager    *common.DNSConfigManager
}

//DefaultDNSNsmMonitor creates default DNS nsm monitor
func DefaultDNSNsmMonitor() NSMApp {
	return NewDNSNsmMonitor(DefaultPathToCorefile, DefaultReloadCorefileTime)
}

//NewDNSNsmMonitor creates new dns nsm monitor with a specific path to corefile and time to reload corefile
func NewDNSNsmMonitor(pathToCorefile string, reloadTime time.Duration) NSMApp {
	dnsConfigManager := common.NewDNSConfigManager(defaultBasicDNSConfig(), reloadTime)
	corefile := dnsConfigManager.Caddyfile(pathToCorefile)
	err := corefile.Save()
	if err != nil {
		logrus.Errorf("An error during initial saving the Corefile: %v, err: %v", corefile.String(), err.Error())
	}
	result := NewNSMMonitorApp()
	result.SetHandler(&NsmDNSMonitorHandler{
		updateCorefileState: notScheduled,
		pathToCorefile:      pathToCorefile,
		dnsConfigManager:    dnsConfigManager,
	})
	return result
}

//Connected checks connection and l handle all dns configs
func (h *NsmDNSMonitorHandler) Connected(conns map[string]*connection.Connection) {
	for _, conn := range conns {
		if conn.Context == nil {
			continue
		}
		if conn.Context.DnsContext == nil {
			continue
		}
		for _, config := range conn.Context.DnsContext.Configs {
			logrus.Infof("Adding dns config with id: %v, value: %v", conn.Id, config)
			h.dnsConfigManager.Store(conn.Id, *config)
		}
	}
	h.scheduleUpdateCorefile()
}

//Closed removes all dns configs related to connection ID
func (h *NsmDNSMonitorHandler) Closed(conn *connection.Connection) {
	logrus.Infof("Deleting config with id %v", conn.Id)
	h.dnsConfigManager.Delete(conn.Id)
	h.scheduleUpdateCorefile()
}

func (h *NsmDNSMonitorHandler) scheduleUpdateCorefile() {
	if !atomic.CompareAndSwapInt32(&h.updateCorefileState, notScheduled, running) {
		return
	}
	go func() {
		defer atomic.StoreInt32(&h.updateCorefileState, notScheduled)
		logrus.Infof("Start to update corefile...")
		corefile := h.dnsConfigManager.Caddyfile(h.pathToCorefile)
		err := corefile.Save()
		if err != nil {
			logrus.Errorf("An error during saving the Corefile: %v, error: %v", corefile.String(), err.Error())
		} else {
			logrus.Infof("Corefile updated")
		}
	}()
}

func defaultBasicDNSConfig() connectioncontext.DNSConfig {
	return connectioncontext.DNSConfig{
		DnsServerIps:  []string{defaultK8sDNSServer},
		SearchDomains: []string{},
	}
}
