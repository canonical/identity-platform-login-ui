package logging

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/canonical/identity_platform_login_ui/internal/http_meta"
)

const version = "master"
const audience = "application"
const service_name = "Identity Platform Login UI"

type logEntry struct {
	Audience       string    `json:"audience"`
	Level          string    `json:"level"`
	Msg            string    `json:"msg"`
	ServiceName    string    `json:"service_name"`
	ServiceVersion string    `json:"service_version"`
	Time           string    `json:"time"`
	Error          *logError `json:"error,omitempty"`
}

type logError struct {
	Debug      string `json:"debug,omitempty"`
	Message    string `json:"message"`
	Reason     string `json:"reason,omitempty"`
	Status     string `json:"status,omitempty"`
	StatusCode int    `json:"status_code,omitempty"`
}

type httpLogEntry struct {
	Level        string       `json:"level"`
	Msg          string       `json:"msg"`
	Time         string       `json:"time"`
	HTTPRequest  httpRequest  `json:"http_request"`
	HTTPResponse httpResponse `json:"http_response"`
	Error        *logError    `json:"error,omitempty"`
}

type httpRequest struct {
	Headers map[string]string `json:"headers"`
	Host    string            `json:"host"`
	Method  string            `json:"method"`
	Path    string            `json:"path"`
	Query   url.Values        `json:"query"`
	Remote  string            `json:"remote"`
	Scheme  string            `json:"scheme"`
}

type httpResponse struct {
	Headers    map[string]string `json:"headers"`
	Size       int               `json:"size"`
	Status     int               `json:"status"`
	TextStatus string            `json:"text_status"`
	Took       int               `json:"took"`
}

func newLogError(debug, msg, reason string) logError {
	return logError{
		Debug:   debug,
		Message: msg,
		Reason:  reason,
	}
}

func (lerr logError) HTTPLogError(statusCode int) logError {
	lerr.StatusCode = statusCode
	lerr.Status = http.StatusText(statusCode)
	return lerr
}

func (l logEntry) Log() {
	lJson, _ := json.Marshal(l)
	fmt.Printf("%s\n", lJson)
}

func (l logEntry) String() string {
	lJson, _ := json.Marshal(l)
	return string(lJson)
}

func (l logError) String() string {
	lJson, _ := json.Marshal(l)
	return string(lJson)
}

func (l httpRequest) String() string {
	lJson, _ := json.Marshal(l)
	return string(lJson)
}

func (l httpResponse) String() string {
	lJson, _ := json.Marshal(l)
	return string(lJson)
}

func (l httpLogEntry) Log() {
	lJson, _ := json.Marshal(l)
	fmt.Printf("%s\n", lJson)
}

func (l httpLogEntry) String() string {
	lJson, _ := json.Marshal(l)
	return string(lJson)
}

func defaultLogEntry() logEntry {
	return logEntry{
		Audience:       audience,
		ServiceName:    service_name,
		ServiceVersion: version,
		Time:           time.Now().Format("2023-06-14T17:19:02.681647104Z"),
	}
}

func defaultHTTPLogEntry(w http.ResponseWriter, r *http.Request) httpLogEntry {
	requestHeaders := make(map[string]string)
	for k := range r.Header {
		requestHeaders[k] = r.Header.Get(k)
	}
	responseHeaders := make(map[string]string)
	for k := range w.Header() {
		responseHeaders[k] = w.Header().Get(k)
	}
	req := httpRequest{
		Headers: requestHeaders,
		Host:    r.Host,
		Method:  r.Method,
		Path:    r.URL.Path,
		Query:   r.URL.Query(),
		Remote:  r.RemoteAddr,
		Scheme:  "http",
	}

	resp := httpResponse{
		Headers:    responseHeaders,
		Size:       http_meta.GetResponseSize(w),
		Status:     http_meta.GetResponseStatus(w),
		TextStatus: http.StatusText(http_meta.GetResponseStatus(w)),
	}
	return httpLogEntry{
		Time:         time.Now().Format("2023-06-14T17:19:02.681647104Z"),
		HTTPRequest:  req,
		HTTPResponse: resp,
	}
}

func InfoLogEntry(msg string) logEntry {
	le := defaultLogEntry()
	le.Level = "info"
	le.Msg = msg
	return le
}

func WarnLogEntry(msg string) logEntry {
	le := defaultLogEntry()
	le.Level = "warning"
	le.Msg = msg
	return le
}

func ErrorLogEntry(msg string) logEntry {
	le := defaultLogEntry()
	le.Level = "error"
	le.Msg = msg
	return le
}

func InfoHTTPLogEntry(w http.ResponseWriter, r *http.Request, msg string) httpLogEntry {
	he := defaultHTTPLogEntry(w, r)
	he.Level = "info"
	he.Msg = msg
	return he
}

func WarnHTTPLogEntry(w http.ResponseWriter, r *http.Request, msg string) httpLogEntry {
	he := defaultHTTPLogEntry(w, r)
	he.Level = "warning"
	he.Msg = msg
	return he
}

func ErrorHTTPLogEntry(w http.ResponseWriter, r *http.Request, msg string) httpLogEntry {
	he := defaultHTTPLogEntry(w, r)
	he.Level = "error"
	he.Msg = msg
	return he
}

func (le logEntry) WithError(debug, msg, reason string) logEntry {
	e := newLogError(debug, msg, reason)
	le.Error = &e
	return le
}

func (le logEntry) WithHTTPError(debug, msg, reason string, statusCode int) logEntry {
	e := newLogError(debug, msg, reason).HTTPLogError(statusCode)
	le.Error = &e
	return le
}

func (he httpLogEntry) WithHTTPError(debug, msg, reason string) httpLogEntry {
	e := newLogError(debug, msg, reason).HTTPLogError(he.HTTPResponse.Status)
	he.Error = &e
	return he
}

// Middleware Logs http request.
func Middleware(next http.HandlerFunc) func(rw http.ResponseWriter, r *http.Request) {
	return func(rw http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(rw, r)
		status := http_meta.GetResponseStatus(rw)
		path := r.URL.Path
		if status == http.StatusOK {
			InfoHTTPLogEntry(rw, r, "completed handling request").Log()
			return
		}

		if strings.Contains(path, "self-service") || strings.Contains(path, "api/consent") {
			WarnHTTPLogEntry(rw, r, "issue with proxied request").WithHTTPError("", fmt.Sprintf("issue with proxying for endpoint %s", path), "").Log()
			return
		} else {
			ErrorHTTPLogEntry(rw, r, "issue with UI request").WithHTTPError("", fmt.Sprintf("issue with endpoint %s", path), "").Log()
			return
		}
	}
}
