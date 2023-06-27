package prometheus

import (
	"fmt"

	"github.com/canonical/identity_platform_login_ui/internal/logging"
	"github.com/canonical/identity_platform_login_ui/internal/monitoring"
	"github.com/prometheus/client_golang/prometheus"
)

type Monitor struct {
	service string

	responseTime *prometheus.HistogramVec

	logger logging.LoggerInterface
}

func (m *Monitor) GetService() string {
	return m.service
}

func (m *Monitor) GetResponseTimeMetric(tags map[string]string) (monitoring.MetricInterface, error) {
	if m.responseTime == nil {
		return nil, fmt.Errorf("metric not instantiated")
	}

	return m.responseTime.With(tags), nil
}

func (m *Monitor) registerHistograms() {
	histograms := make([]*prometheus.HistogramVec, 0)

	labels := map[string]string{
		"service": m.service,
	}

	m.responseTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:        "http_response_time_seconds",
			Help:        "http_response_time_seconds",
			ConstLabels: labels,
		},
		[]string{"route", "status"},
	)

	histograms = append(histograms, m.responseTime)

	for _, histogram := range histograms {
		err := prometheus.Register(histogram)

		switch err.(type) {
		case nil:
			return
		case prometheus.AlreadyRegisteredError:
			m.logger.Debugf("metric %v already registered", histogram)
		default:
			m.logger.Errorf("metric %v could not be registered", histogram)
		}
	}
}

func NewMonitor(service string, logger logging.LoggerInterface) *Monitor {
	m := new(Monitor)

	m.service = service
	m.logger = logger

	m.registerHistograms()

	return m
}
