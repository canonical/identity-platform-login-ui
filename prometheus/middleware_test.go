package prometheus

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/assert"
)

const middleware_testApp = "middleware-test"
const middleware_testPath = "/middleware-test"

func TestMetricsManagerGetLabelForPath(t *testing.T) {
	t.Run("case=no-endpoint-registered", func(t *testing.T) {
		mm := NewMetricsManagerWithPrefix(middleware_testApp, "http", "", "", "")
		t.Cleanup(cleanup(mm))
		r := httptest.NewRequest("GET", middleware_testPath, strings.NewReader(""))
		assert.Equal(t, "{unmatched}", mm.getLabelForPath(r))
	})

	t.Run("case=registered-routers-match-no-params", func(t *testing.T) {
		mm := NewMetricsManagerWithPrefix(middleware_testApp, "http", "", "", "")
		t.Cleanup(cleanup(mm))
		mm.RegisterRoutes(middleware_testPath)
		r := httptest.NewRequest("GET", middleware_testPath, strings.NewReader(""))
		assert.Equal(t, middleware_testPath, mm.getLabelForPath(r))
	})
}

func TestMiddleware(t *testing.T) {
	mm := NewMetricsManagerWithPrefix(middleware_testApp, "http", "", "", "")
	t.Cleanup(cleanup(mm))
	mm.RegisterRoutes(PrometheusPath, middleware_testPath)

	req := httptest.NewRequest(http.MethodGet, middleware_testPath, nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mm.Middleware(testHandler)(w, req)
	resp := w.Result()
	assert.Equalf(t, http.StatusBadRequest, resp.StatusCode, "Expected %s, got %s", http.StatusBadRequest, resp.StatusCode)

	req = httptest.NewRequest(http.MethodGet, PrometheusPath, nil)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	mm.Middleware(PrometheusMetrics)(w, req)
	resp = w.Result()

	textParser := expfmt.TextParser{}
	text, err := textParser.TextToMetricFamilies(resp.Body)
	assert.Nilf(t, err, "Expected nil, got %s", err)
	assert.Equal(t, "http_response_time_seconds", *text["http_response_time_seconds"].Name)
	assert.Equal(t, middleware_testPath, getLabelValue("endpoint", text["http_response_time_seconds"].Metric))
	assert.Equal(t, middleware_testApp, getLabelValue("app", text["http_response_time_seconds"].Metric))

	assert.Equal(t, "http_requests_total", *text["http_requests_total"].Name)
	assert.Equal(t, "400", getLabelValue("code", text["http_requests_total"].Metric))
	assert.Equal(t, middleware_testPath, getLabelValue("endpoint", text["http_requests_total"].Metric))
	assert.Equal(t, middleware_testApp, getLabelValue("app", text["http_requests_total"].Metric))

	assert.Equal(t, "http_requests_duration_seconds", *text["http_requests_duration_seconds"].Name)
	assert.Equal(t, "400", getLabelValue("code", text["http_requests_duration_seconds"].Metric))
	assert.Equal(t, middleware_testPath, getLabelValue("endpoint", text["http_requests_duration_seconds"].Metric))
	assert.Equal(t, middleware_testApp, getLabelValue("app", text["http_requests_duration_seconds"].Metric))

	assert.Equal(t, "http_response_size_bytes", *text["http_response_size_bytes"].Name)
	assert.Equal(t, "400", getLabelValue("code", text["http_response_size_bytes"].Metric))
	assert.Equal(t, middleware_testApp, getLabelValue("app", text["http_response_size_bytes"].Metric))

	assert.Equal(t, "http_requests_size_bytes", *text["http_requests_size_bytes"].Name)
	assert.Equal(t, "400", getLabelValue("code", text["http_requests_size_bytes"].Metric))
	assert.Equal(t, middleware_testApp, getLabelValue("app", text["http_requests_size_bytes"].Metric))

	assert.Equal(t, "http_requests_statuses_total", *text["http_requests_statuses_total"].Name)
	assert.Equal(t, "4xx", getLabelValue("status_bucket", text["http_requests_statuses_total"].Metric))
	assert.Equal(t, middleware_testApp, getLabelValue("app", text["http_requests_statuses_total"].Metric))
}
