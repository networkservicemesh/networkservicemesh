package sidecars

import (
	"time"

	"github.com/networkservicemesh/networkservicemesh/utils"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
)

const (
	//DefaultPathToCorefile contains default path to Corefile
	DefaultPathToCorefile = "/etc/coredns/Corefile"
	//DefaultReloadCorefileTime means time to reload Corefile
	DefaultReloadCorefileTime = time.Second * 5
	defaultK8sDNSServer       = "10.96.0.10"
)

//nsmDNSMonitorHandler implements NSMMonitorHandler interface for handling dnsConfigs
type nsmDNSMonitorHandler struct {
	EmptyNSMMonitorHandler
	dnsConfigManager *utils.DNSConfigManager
	corefileUpdater  utils.Operation
}

//NewNsmDNSMonitorHandler creates new DNS monitor handler
func NewNsmDNSMonitorHandler(corefilePath string, reloadCorefilePeriod time.Duration) NSMMonitorHandler {
	dnsConfigManager := utils.NewDNSConfigManager(defaultBasicDNSConfig(), reloadCorefilePeriod)
	corefileUpdater := utils.NewSingleAsyncOperation(func() {
		file := dnsConfigManager.Caddyfile(corefilePath)
		err := file.Save()
		if err != nil {
			logrus.Errorf("An error during updating corefile: %v", err)
		}
	})
	corefile := dnsConfigManager.Caddyfile(corefilePath)
	err := corefile.Save()
	if err != nil {
		logrus.Errorf("An error during initial saving the Corefile: %v, err: %v", corefile.String(), err.Error())
	}
	return &nsmDNSMonitorHandler{
		dnsConfigManager: dnsConfigManager,
		corefileUpdater:  corefileUpdater,
	}
}

func (h *nsmDNSMonitorHandler) Connected(conns map[string]*connection.Connection) {
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
	h.corefileUpdater.Run()
}

func (h *nsmDNSMonitorHandler) Closed(conn *connection.Connection) {
	logrus.Infof("Deleting config with id %v", conn.Id)
	h.dnsConfigManager.Delete(conn.Id)
	h.corefileUpdater.Run()
}

func defaultBasicDNSConfig() connectioncontext.DNSConfig {
	return connectioncontext.DNSConfig{
		DnsServerIps:  []string{defaultK8sDNSServer},
		SearchDomains: []string{},
	}
}
