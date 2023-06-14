package prometheus

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	ioprometheusclient "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/assert"
)

const metrics_testPath = "/metrics-test"
const metrics_testApp = "metrics-test"

func TestInstrument(t *testing.T) {
	mm := NewMetricsManagerWithPrefix(metrics_testApp, "http", "", "", "")
	t.Cleanup(Cleanup(mm))
	mm.RegisterRoutes(PrometheusPath, metrics_testPath)

	req := httptest.NewRequest(http.MethodGet, metrics_testPath, nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ResponseWithStatusCodeMiddleware(mm.prometheusMetrics.Instrument(w, testHandler, mm.getLabelForPath(req)))(w, req)
	resp := w.Result()
	assert.Equalf(t, http.StatusBadRequest, resp.StatusCode, "Expected %s, got %s", http.StatusBadRequest, resp.StatusCode)

	req = httptest.NewRequest(http.MethodGet, PrometheusPath, nil)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	ResponseWithStatusCodeMiddleware(mm.prometheusMetrics.Instrument(w, PrometheusMetrics, mm.getLabelForPath(req)))(w, req)
	resp = w.Result()

	textParser := expfmt.TextParser{}
	text, err := textParser.TextToMetricFamilies(resp.Body)
	assert.Nilf(t, err, "Expected nil, got %s", err)
	assert.Equal(t, "http_response_time_seconds", *text["http_response_time_seconds"].Name)
	assert.Equal(t, metrics_testPath, getLabelValue("endpoint", text["http_response_time_seconds"].Metric))
	assert.Equal(t, metrics_testApp, getLabelValue("app", text["http_response_time_seconds"].Metric))

	assert.Equal(t, "http_requests_total", *text["http_requests_total"].Name)
	assert.Equal(t, "400", getLabelValue("code", text["http_requests_total"].Metric))
	assert.Equal(t, metrics_testPath, getLabelValue("endpoint", text["http_requests_total"].Metric))
	assert.Equal(t, metrics_testApp, getLabelValue("app", text["http_requests_total"].Metric))

	assert.Equal(t, "http_requests_duration_seconds", *text["http_requests_duration_seconds"].Name)
	assert.Equal(t, "400", getLabelValue("code", text["http_requests_duration_seconds"].Metric))
	assert.Equal(t, metrics_testPath, getLabelValue("endpoint", text["http_requests_duration_seconds"].Metric))
	assert.Equal(t, metrics_testApp, getLabelValue("app", text["http_requests_duration_seconds"].Metric))

	assert.Equal(t, "http_response_size_bytes", *text["http_response_size_bytes"].Name)
	assert.Equal(t, "400", getLabelValue("code", text["http_response_size_bytes"].Metric))
	assert.Equal(t, metrics_testApp, getLabelValue("app", text["http_response_size_bytes"].Metric))

	assert.Equal(t, "http_requests_size_bytes", *text["http_requests_size_bytes"].Name)
	assert.Equal(t, "400", getLabelValue("code", text["http_requests_size_bytes"].Metric))
	assert.Equal(t, metrics_testApp, getLabelValue("app", text["http_requests_size_bytes"].Metric))

	assert.Equal(t, "http_requests_statuses_total", *text["http_requests_statuses_total"].Name)
	assert.Equal(t, "4xx", getLabelValue("status_bucket", text["http_requests_statuses_total"].Metric))
	assert.Equal(t, metrics_testApp, getLabelValue("app", text["http_requests_statuses_total"].Metric))
}

func getLabelValue(name string, metric []*ioprometheusclient.Metric) string {
	for _, label := range metric[0].Label {
		if *label.Name == name {
			return *label.Value
		}
	}

	return ""
}

func testHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusBadRequest)
	w.Header().Set("Content-Type", "application/json")
	resp := make(map[string]string)
	resp["message"] = http.StatusText(http.StatusBadRequest)
	jsonResp, _ := json.Marshal(resp)
	w.Write(jsonResp)
	return
}
