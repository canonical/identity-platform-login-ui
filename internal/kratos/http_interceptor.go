package kratos

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

type InterceptorFactory func(http.RoundTripper) http.RoundTripper

type methodOnly struct {
	Method string `json:"method"`
}

type HTTPIdentifierFirstInterceptor struct {
	proxy http.RoundTripper
}

func (i *HTTPIdentifierFirstInterceptor) RoundTrip(req *http.Request) (*http.Response, error) {

	// req will be a POST to :4433/self-service/login?flow=<flowId> with "method" == "identifier_first"
	if req.Method == http.MethodPost && strings.HasPrefix(req.URL.Path, "/self-service/login") {
		var body methodOnly

		bodyBytes, _ := io.ReadAll(req.Body)
		defer req.Body.Close()

		// replace the body that was consumed
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))

		_ = json.Unmarshal(bodyBytes, &body)

		if body.Method == "identifier_first" {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
	}

	return i.proxy.RoundTrip(req)
}

func NewHTTPInterceptor(proxy http.RoundTripper) http.RoundTripper {
	return &HTTPIdentifierFirstInterceptor{
		proxy: proxy,
	}
}
