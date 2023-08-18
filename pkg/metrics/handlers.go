package metrics

import (
	"net/http"

	"github.com/canonical/identity-platform-login-ui/internal/http_meta"
	"github.com/canonical/identity-platform-login-ui/internal/logging"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type API struct {
	logger logging.LoggerInterface
}

func (a *API) RegisterEndpoints(mux http_meta.RestInterface) {
	mux.Get("/api/v0/metrics", a.prometheusHTTP)
}

func (a *API) prometheusHTTP(w http.ResponseWriter, r *http.Request) {
	promhttp.Handler().ServeHTTP(w, r)
}

func NewAPI(logger logging.LoggerInterface) *API {
	a := new(API)

	a.logger = logger

	return a
}
