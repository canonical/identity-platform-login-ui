package monitoring

type MonitorInterface interface {
	GetService() string
	GetResponseTimeMetric(map[string]string) (MetricInterface, error)
	RegisterEndpoints(...string)
	VerifyEndpoint(string) bool
}

type MetricInterface interface {
	Observe(float64)
}
