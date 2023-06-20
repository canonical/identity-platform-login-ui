package monitoring

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/golang/mock/gomock"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stretchr/testify/assert"
)

//go:generate mockgen -build_flags=--mod=mod -package monitoring -destination ./mock_monitor.go -source=./interfaces.go
//go:generate mockgen -build_flags=--mod=mod -package monitoring -destination ./mock_logger.go -source=../logging/interfaces.go

type API struct{}

func (a *API) RegisterEndpoints(router *chi.Mux) {
	router.Get("/api/v1/metrics", a.prometheusHTTP)
	router.Get("/api/test", a.test)
}

func (a *API) prometheusHTTP(w http.ResponseWriter, r *http.Request) {
	promhttp.Handler().ServeHTTP(w, r)
}

func (a *API) test(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

func TestMiddlewareResponseTime(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMonitor := NewMockMonitorInterface(ctrl)
	mockMetric := NewMockMetricInterface(ctrl)
	mockLogger := NewMockLoggerInterface(ctrl)
	mockMonitor.EXPECT().GetService().Times(1)
	mockMonitor.EXPECT().GetResponseTimeMetric(gomock.Any()).Times(1).Return(mockMetric, nil)
	mockMetric.EXPECT().Observe(gomock.Any()).Times(1)

	assert := assert.New(t)

	router := chi.NewMux()

	router.Use(NewMiddleware(mockMonitor, mockLogger).ResponseTime())

	new(API).RegisterEndpoints(router)

	// setup metrics endpoint
	req, err := http.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Content-Type", "application/json")
	assert.Nil(err, "error should be nil")

	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)
}
