package metrics

type Statistics struct {
	Name       string
	Metrics map[string]string
}

type MetricsMonitor interface {
	HandleMetrics(statistics *Statistics)
}
