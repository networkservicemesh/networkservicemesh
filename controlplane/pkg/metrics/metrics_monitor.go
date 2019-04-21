package metrics

import "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"

type MetricsMonitor interface {
	HandleMetrics(statistics map[string]*crossconnect.Metrics)
}
