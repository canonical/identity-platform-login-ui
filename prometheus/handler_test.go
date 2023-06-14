package prometheus

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/assert"
)

const handler_testPath = "/handler-test"
const handler_testApp = "handler-test"

func TestPrometheusHandler(t *testing.T) {
	mm := NewMetricsManagerWithPrefix(handler_testApp, "http", "", "", "")
	t.Cleanup(Cleanup(mm))
	mm.RegisterRoutes(PrometheusPath)

	req := httptest.NewRequest(http.MethodGet, handler_testPath, nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mm.Middleware(PrometheusMetrics)(w, req)
	resp := w.Result()
	assert.Equalf(t, http.StatusOK, resp.StatusCode, "Expected %s, got %s", http.StatusOK, resp.StatusCode)
	textParser := expfmt.TextParser{}
	text, err := textParser.TextToMetricFamilies(resp.Body)
	assert.Nilf(t, err, "Expected nil, got %s", err)
	assert.Equalf(t, "go_info", *text["go_info"].Name, "Expected %s, got %s", "go_info", *text["go_info"].Name)
}
