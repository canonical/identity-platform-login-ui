package logging

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/canonical/identity_platform_login_ui/http_meta"

	"github.com/stretchr/testify/assert"
)

const testMessage = "this is a test"
const testPath = "/test"
const testHeaderKey = "Test_key"
const testHeaderValue = "test_value"

func keyValueHelper(k, v string) string {
	return fmt.Sprintf("\"%s\":\"%s\"", k, v)
}

func _TestHelper(t *testing.T, testref, field string) {
	assert.Truef(t, strings.Contains(testref, field), "Expected %s to be included", field)
}

func AssertLogEntryBasics(t *testing.T, testref string) {
	_TestHelper(t, testref, keyValueHelper("audience", "application"))
	_TestHelper(t, testref, keyValueHelper("service_name", "Identity Platform Login UI"))
	_TestHelper(t, testref, keyValueHelper("service_version", "master"))
}

func AssertHTTPLogEntryBasics(t *testing.T, testref, path string) {
	_TestHelper(t, testref, "http_request")
	_TestHelper(t, testref, "http_response")
	_TestHelper(t, testref, keyValueHelper("path", path))
}

func AssertLogLevel(t *testing.T, testref string, level string) {
	_TestHelper(t, testref, keyValueHelper("level", level))
}

func AssertLogMessage(t *testing.T, testref string, msg string) {
	_TestHelper(t, testref, keyValueHelper("msg", msg))
}

func AssertLogErrorMessage(t *testing.T, testref string, msg string) {
	_TestHelper(t, testref, keyValueHelper("message", msg))
}

func AssertLogErrorStatus(t *testing.T, testref string, status int) {
	_TestHelper(t, testref, fmt.Sprintf("\"status_code\":%d", status))
	_TestHelper(t, testref, keyValueHelper("status", http.StatusText(status)))
}

func AssertHeader(t *testing.T, testref, key, value string) {
	_TestHelper(t, testref, keyValueHelper(key, value))
}

func AssertHTTPResponse(t *testing.T, testref string, status, size int) {
	_TestHelper(t, testref, fmt.Sprintf("\"status\":%d", status))
	_TestHelper(t, testref, fmt.Sprintf("\"size\":%d", size))
	_TestHelper(t, testref, keyValueHelper("text_status", http.StatusText(status)))
}

func TestInfoLogEntry(t *testing.T) {
	testString := InfoLogEntry(testMessage).String()
	t.Logf("%s\n", testString)
	AssertLogEntryBasics(t, testString)
	AssertLogLevel(t, testString, "info")
	AssertLogMessage(t, testString, testMessage)
}

func TestWarnLogEntry(t *testing.T) {
	testString := WarnLogEntry(testMessage).String()
	t.Logf("%s\n", testString)
	AssertLogEntryBasics(t, testString)
	AssertLogLevel(t, testString, "warning")
	AssertLogMessage(t, testString, testMessage)
}

func TestWarnLogEntryWithError(t *testing.T) {
	entry := WarnLogEntry(testMessage).WithError("", testMessage, "")
	testString := entry.String()
	errorString := entry.Error.String()
	t.Logf("%s\n", testString)
	AssertLogEntryBasics(t, testString)
	AssertLogLevel(t, testString, "warning")
	AssertLogMessage(t, testString, testMessage)
	AssertLogErrorMessage(t, errorString, testMessage)
}

func TestErrorLogEntryWithError(t *testing.T) {
	entry := ErrorLogEntry(testMessage).WithError("", testMessage, "")
	testString := entry.String()
	errorString := entry.Error.String()
	t.Logf("%s\n", testString)
	AssertLogEntryBasics(t, testString)
	AssertLogLevel(t, testString, "error")
	AssertLogMessage(t, testString, testMessage)
	AssertLogErrorMessage(t, errorString, testMessage)
}

func TestErrorLogEntryWithHTTPError(t *testing.T) {
	entry := ErrorLogEntry(testMessage).WithHTTPError("", testMessage, "", 500)
	testString := entry.String()
	errorString := entry.Error.String()
	t.Logf("%s\n", testString)
	AssertLogEntryBasics(t, testString)
	AssertLogLevel(t, testString, "error")
	AssertLogMessage(t, testString, testMessage)
	AssertLogErrorMessage(t, errorString, testMessage)
	AssertLogErrorStatus(t, errorString, 500)
}

func TestInfoHTTPLogEntry(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, testPath, nil)
	req.Header.Set(testHeaderKey, testHeaderValue)
	w := http_meta.NewresponseWriterMeta(httptest.NewRecorder())
	http_meta.ResponseWriterMetaMiddleware(test200Handler)(w, req)

	testString := InfoHTTPLogEntry(w, req, testMessage).String()
	t.Logf("%s\n", testString)
	AssertHTTPLogEntryBasics(t, testString, testPath)
	AssertLogLevel(t, testString, "info")
	AssertLogMessage(t, testString, testMessage)
	AssertHeader(t, testString, testHeaderKey, testHeaderValue)
	AssertHTTPResponse(t, testString, 200, len("\"message\":\"OK\"")+2)
}

func TestWarnHTTPLogEntry(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, testPath, nil)
	req.Header.Set(testHeaderKey, testHeaderValue)
	w := http_meta.NewresponseWriterMeta(httptest.NewRecorder())
	http_meta.ResponseWriterMetaMiddleware(test200Handler)(w, req)

	testString := WarnHTTPLogEntry(w, req, testMessage).String()
	t.Logf("%s\n", testString)
	AssertHTTPLogEntryBasics(t, testString, testPath)
	AssertLogLevel(t, testString, "warning")
	AssertLogMessage(t, testString, testMessage)
	AssertHeader(t, testString, testHeaderKey, testHeaderValue)
	AssertHTTPResponse(t, testString, 200, len("\"message\":\"OK\"")+2)
}

func TestWarnHTTPLogEntryWithHTTPError(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, testPath, nil)
	req.Header.Set(testHeaderKey, testHeaderValue)
	w := http_meta.NewresponseWriterMeta(httptest.NewRecorder())
	http_meta.ResponseWriterMetaMiddleware(test200Handler)(w, req)

	entry := WarnHTTPLogEntry(w, req, testMessage).WithHTTPError("", testMessage, "")
	testString := entry.String()
	errorString := entry.Error.String()

	t.Logf("%s\n", testString)
	AssertHTTPLogEntryBasics(t, testString, testPath)
	AssertLogLevel(t, testString, "warning")
	AssertLogMessage(t, testString, testMessage)
	AssertHeader(t, testString, testHeaderKey, testHeaderValue)
	AssertHTTPResponse(t, testString, 200, len("\"message\":\"OK\"")+2)
	AssertLogErrorMessage(t, errorString, testMessage)
	AssertLogErrorStatus(t, errorString, 200)
}

func TestErrorHTTPLogEntryWithHTTPError(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, testPath, nil)
	req.Header.Set(testHeaderKey, testHeaderValue)
	w := http_meta.NewresponseWriterMeta(httptest.NewRecorder())
	http_meta.ResponseWriterMetaMiddleware(test200Handler)(w, req)

	entry := ErrorHTTPLogEntry(w, req, testMessage).WithHTTPError("", testMessage, "")
	testString := entry.String()
	errorString := entry.Error.String()

	t.Logf("%s\n", testString)
	AssertHTTPLogEntryBasics(t, testString, testPath)
	AssertLogLevel(t, testString, "error")
	AssertLogMessage(t, testString, testMessage)
	AssertHeader(t, testString, testHeaderKey, testHeaderValue)
	AssertHTTPResponse(t, testString, 200, len("\"message\":\"OK\"")+2)
	AssertLogErrorMessage(t, errorString, testMessage)
	AssertLogErrorStatus(t, errorString, 200)
}

func test200Handler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	resp := make(map[string]string)
	resp["message"] = http.StatusText(http.StatusOK)
	jsonResp, _ := json.Marshal(resp)
	w.Write(jsonResp)
	return
}
