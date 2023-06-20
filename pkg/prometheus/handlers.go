package prometheus

import (
	"net/http"

	"github.com/canonical/identity_platform_login_ui/internal/logging"
	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type API struct {
	logger logging.LoggerInterface
}

func (a *API) RegisterEndpoints(mux *chi.Mux) {
	mux.Get("/metrics/prometheus", a.prometheusHTTP)
}

func (a *API) prometheusHTTP(w http.ResponseWriter, r *http.Request) {
	promhttp.Handler().ServeHTTP(w, r)
}

func NewAPI(logger logging.LoggerInterface) *API {
	a := new(API)

	a.logger = logger

	return a
}
