package prometheus

import (
	"fmt"

	"github.com/canonical/identity-platform-login-ui/internal/logging"
	"github.com/prometheus/client_golang/prometheus"
)

type Monitor struct {
	service string

	responseTime           *prometheus.HistogramVec
	dependencyAvailability *prometheus.GaugeVec

	logger logging.LoggerInterface
}

func (m *Monitor) GetService() string {
	return m.service
}

func (m *Monitor) SetResponseTimeMetric(tags map[string]string, value float64) error {
	if m.responseTime == nil {
		return fmt.Errorf("metric not instantiated")
	}

	m.responseTime.With(tags).Observe(value)

	return nil
}

func (m *Monitor) SetDependencyAvailability(tags map[string]string, value float64) error {
	if m.dependencyAvailability == nil {
		return fmt.Errorf("metric not instantiated")
	}

	m.dependencyAvailability.With(tags).Set(value)

	return nil
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

func (m *Monitor) registerGauges() {
	gauges := make([]*prometheus.GaugeVec, 0)

	labels := map[string]string{
		"service": m.service,
	}

	m.dependencyAvailability = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:        "dependency_available",
			Help:        "dependency_available",
			ConstLabels: labels,
		},
		[]string{"component"},
	)

	gauges = append(gauges, m.dependencyAvailability)

	for _, gauge := range gauges {
		err := prometheus.Register(gauge)

		switch err.(type) {
		case nil:
			return
		case prometheus.AlreadyRegisteredError:
			m.logger.Debugf("metric %v already registered", gauge)
		default:
			m.logger.Errorf("metric %v could not be registered", gauge)
		}
	}
}
func NewMonitor(service string, logger logging.LoggerInterface) *Monitor {
	m := new(Monitor)

	m.service = service
	m.logger = logger

	m.registerHistograms()
	m.registerGauges()

	return m
}
