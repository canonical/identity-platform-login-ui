package monitoring

type MonitorInterface interface {
	GetService() string
	GetResponseTimeMetric(map[string]string) (MetricInterface, error)
}

type MetricInterface interface {
	Observe(float64)
}
