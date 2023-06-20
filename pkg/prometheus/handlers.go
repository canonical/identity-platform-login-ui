package prometheus

import (
	"net/http"

	"github.com/canonical/identity_platform_login_ui/internal/logging"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type API struct {
	logger logging.LoggerInterface
}

func (a *API) RegisterEndpoints(mux *http.ServeMux) {
	mux.HandleFunc("/metrics/prometheus", a.prometheusHTTP)
}

func (a *API) prometheusHTTP(w http.ResponseWriter, r *http.Request) {
	promhttp.Handler().ServeHTTP(w, r)
}

func NewAPI(logger logging.LoggerInterface) *API {
	a := new(API)

	a.logger = logger

	return a
}
