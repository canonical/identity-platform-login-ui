package status

import (
	"encoding/json"
	"net/http"

	"github.com/canonical/identity_platform_login_ui/internal/logging"
	"github.com/canonical/identity_platform_login_ui/internal/monitoring"
	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/trace"
)

const okValue = "ok"

type Status struct {
	Status    string     `json:"status"`
	BuildInfo *BuildInfo `json:"buildInfo"`
}

type API struct {
	tracer trace.Tracer

	monitor monitoring.MonitorInterface
	logger  logging.LoggerInterface
}

func (a *API) RegisterEndpoints(mux *chi.Mux) {
	mux.Get("/api/v0/status", a.alive)
	mux.Get("/api/v0/version", a.version)

}

func (a *API) alive(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	rr := Status{
		Status: okValue,
	}

	_, span := a.tracer.Start(r.Context(), "buildInfo")

	if buildInfo := buildInfo(); buildInfo != nil {
		rr.BuildInfo = buildInfo
	}

	span.End()

	json.NewEncoder(w).Encode(rr)

}

func (a *API) version(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	info := new(BuildInfo)
	if buildInfo := buildInfo(); buildInfo != nil {
		info = buildInfo
	}

	json.NewEncoder(w).Encode(info)

}

func NewAPI(tracer trace.Tracer, monitor monitoring.MonitorInterface, logger logging.LoggerInterface) *API {
	a := new(API)

	a.tracer = tracer
	a.monitor = monitor
	a.logger = logger

	return a
}
