package prometheus

const (
	EXPECTED_NIL_ERROR_MESSAGE   = "expected error to be nil got %v"
	HANDLE_CREATE_FLOW_URL       = "/api/kratos/self-service/login/browser?aal=aal1&login_challenge=&refresh=false&return_to=http://test.test"
	COOKIE_NAME                  = "ory_kratos_session"
	COOKIE_VALUE                 = "test-token"
	UPDATE_LOGIN_FLOW_METHOD     = "oidc"
	UPDATE_LOGIN_FLOW_PROVIDER   = "microsoft"
	HANDLE_UPDATE_LOGIN_FLOW_URL = "/api/kratos/self-service/login?flow=1111"
	HANDLE_GET_LOGIN_FLOW_URL    = "/api/kratos/self-service/login/flows?id=1111"
	HANDLE_ERROR_URL             = "/api/kratos/self-service/errors?id=1111"
	HANDLE_CONSENT_URL           = "/api/consent?consent_challenge=test_challange"
	HANDLE_ALIVE_URL             = "/health/alive"
	PROMETHEUS_ENDPOINT          = PrometheusPath
	handler_testPath             = "/handler-test"
	handler_testApp              = "handler-test"
)

// import (
// 	"net/http"
// 	"net/http/httptest"
// 	"testing"

// 	"github.com/prometheus/common/expfmt"
// 	"github.com/stretchr/testify/assert"
// )

// // --------------------------------------------
// // TESTING 	PROMETHEUS INSTRUMENTATION
// // --------------------------------------------
// func TestHandlePrometheusInstrumentation(t *testing.T) {
// 	//test strings
// 	prefix := "http_"
// 	app := "identity-platform-login-ui-operator"
// 	requests_duration_seconds_count_format := "%srequests_duration_seconds_count{app=\"%s\",buildTime=\"\",code=\"200\",endpoint=\"%s\",hash=\"\",method=\"get\",version=\"\"} 1"
// 	requests_size_bytes_count_format := "%srequests_size_bytes_count{app=\"%s\",buildTime=\"\",code=\"200\",hash=\"\",method=\"get\",version=\"\"} 3"
// 	requests_total_format := "%srequests_total{app=\"%s\",buildTime=\"\",code=\"200\",endpoint=\"%s\",hash=\"\",method=\"get\",version=\"\"} 1"
// 	response_size_bytes_count_format := "%sresponse_size_bytes_count{app=\"%s\",buildTime=\"\",code=\"200\",hash=\"\",method=\"get\",version=\"\"} 3"
// 	response_time_seconds_count_format := "%sresponse_time_seconds_count{app=\"%s\",buildTime=\"\",endpoint=\"%s\",hash=\"\",version=\"\"} 1"
// 	metric_handler_requests_total := "promhttp_metric_handler_requests_total{code=\"200\"} 0"

// 	formatHelper := func(handlerURL string) (r1 string, r2 string, r3 string, r4 string, r5 string) {
// 		endpoint, err := url.Parse(handlerURL)
// 		if err != nil {
// 			t.Errorf("expected error to be nil got %v", err)
// 		}
// 		log.Println("Reference URL: " + endpoint.Path)
// 		r1 = fmt.Sprintf(requests_duration_seconds_count_format, prefix, app, endpoint.Path)
// 		r2 = fmt.Sprintf(requests_size_bytes_count_format, prefix, app)
// 		r3 = fmt.Sprintf(requests_total_format, prefix, app, endpoint.Path)
// 		r4 = fmt.Sprintf(response_size_bytes_count_format, prefix, app)
// 		r5 = fmt.Sprintf(response_time_seconds_count_format, prefix, app, endpoint.Path)
// 		return
// 	}

// 	//init clients and middleware
// 	testServers.CreateTestServers(t)
// 	metricsManager := setUpPrometheus()
// 	t.Cleanup(prometheus.Cleanup(metricsManager))

// 	//make requests to different urls
// 	//create flow
// 	req := httptest.NewRequest(http.MethodGet, HANDLE_CREATE_FLOW_URL, nil)
// 	req.Header.Set("Content-Type", "application/json")
// 	w := httptest.NewRecorder()
// 	http_meta.ResponseWriterMetaMiddleware(metricsManager.Middleware(handleCreateFlow))(w, req)
// 	//self-service errors
// 	req = httptest.NewRequest(http.MethodGet, HANDLE_ERROR_URL, nil)
// 	w = httptest.NewRecorder()
// 	http_meta.ResponseWriterMetaMiddleware(metricsManager.Middleware(handleKratosError))(w, req)
// 	//handle consent
// 	req = httptest.NewRequest(http.MethodGet, HANDLE_CONSENT_URL, nil)
// 	w = httptest.NewRecorder()
// 	http_meta.ResponseWriterMetaMiddleware(metricsManager.Middleware(handleConsent))(w, req)

// 	//make request to prometheus endpoint and evaluate test
// 	req = httptest.NewRequest(http.MethodGet, PROMETHEUS_ENDPOINT, nil)
// 	w = httptest.NewRecorder()
// 	http_meta.ResponseWriterMetaMiddleware(metricsManager.Middleware(prometheus.PrometheusMetrics))(w, req)
// 	res := w.Result()
// 	defer res.Body.Close()
// 	data, err := ioutil.ReadAll(res.Body)
// 	if err != nil {
// 		t.Errorf("expected error to be nil got %v", err)
// 	}
// 	dataStrings := strings.Split(string(data), "\n")

// 	assertHelper := func(compareStrings ...string) {
// 		for _, comparison := range compareStrings {
// 			assert.Containsf(t, dataStrings, comparison, "Error in test: Reference string of invalid value: %s", comparison)
// 		}
// 	}

// 	assertHelper(formatHelper(HANDLE_CREATE_FLOW_URL))
// 	assertHelper(formatHelper(HANDLE_ERROR_URL))
// 	assertHelper(formatHelper(HANDLE_CONSENT_URL))
// 	assert.Containsf(t, dataStrings, metric_handler_requests_total, "Error in test: Reference string of invalid value: %s", metric_handler_requests_total)
// }

// func TestPrometheusHandler(t *testing.T) {
// 	mm := NewMetricsManagerWithPrefix(handler_testApp, "http", "", "", "")
// 	t.Cleanup(Cleanup(mm))
// 	mm.RegisterRoutes(PrometheusPath)

// 	req := httptest.NewRequest(http.MethodGet, handler_testPath, nil)
// 	req.Header.Set("Content-Type", "application/json")
// 	w := httptest.NewRecorder()
// 	mm.Middleware(PrometheusMetrics)(w, req)
// 	resp := w.Result()
// 	assert.Equalf(t, http.StatusOK, resp.StatusCode, "Expected %s, got %s", http.StatusOK, resp.StatusCode)
// 	textParser := expfmt.TextParser{}
// 	text, err := textParser.TextToMetricFamilies(resp.Body)
// 	assert.Nilf(t, err, "Expected nil, got %s", err)
// 	assert.Equalf(t, "go_info", *text["go_info"].Name, "Expected %s, got %s", "go_info", *text["go_info"].Name)
// }
