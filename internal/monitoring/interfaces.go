package monitoring

type MonitorInterface interface {
	GetService() string
	SetResponseTimeMetric(map[string]string, float64) error
	SetDependencyAvailability(map[string]string, float64) error
}
