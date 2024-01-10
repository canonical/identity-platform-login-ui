package monitoring

import (
	"github.com/canonical/identity-platform-login-ui/internal/logging"
)

type NoopMonitor struct {
	service string

	logger logging.LoggerInterface
}

func NewNoopMonitor(service string, logger logging.LoggerInterface) *NoopMonitor {
	m := new(NoopMonitor)
	m.service = service
	m.logger = logger
	return m
}

func (m *NoopMonitor) GetService() string {
	return m.service
}
func (m *NoopMonitor) SetResponseTimeMetric(map[string]string, float64) error {
	return nil
}
func (m *NoopMonitor) SetDependencyAvailability(map[string]string, float64) error {
	return nil
}
