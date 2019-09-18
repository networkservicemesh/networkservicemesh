package metrics

import "github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"

type MetricsMonitor interface {
	HandleMetrics(statistics map[string]*crossconnect.Metrics)
}
