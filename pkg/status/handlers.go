package status

import (
	"encoding/json"
	"net/http"

	"github.com/canonical/identity-platform-login-ui/internal/logging"
	"github.com/canonical/identity-platform-login-ui/internal/monitoring"
	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/trace"
)

const okValue = "ok"

type Status struct {
	Status    string     `json:"status"`
	BuildInfo *BuildInfo `json:"buildInfo"`
}

type Health struct {
	Kratos bool `json:"kratos"`
	Hydra  bool `json:"hydra"`
}

type API struct {
	service ServiceInterface

	tracer trace.Tracer

	monitor monitoring.MonitorInterface
	logger  logging.LoggerInterface
}

func (a *API) RegisterEndpoints(mux *chi.Mux) {
	mux.Get("/api/v0/status", a.alive)
	mux.Get("/api/v0/version", a.version)
	mux.Get("/api/v0/health", a.health)
}

func (a *API) alive(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	rr := Status{
		Status: okValue,
	}

	_, span := a.tracer.Start(r.Context(), "status.API.alive")

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

func (a *API) health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	health := new(Health)
	health.Hydra = a.service.HydraStatus(r.Context())
	health.Kratos = a.service.KratosStatus(r.Context())

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(health)
}

func NewAPI(service ServiceInterface, tracer trace.Tracer, monitor monitoring.MonitorInterface, logger logging.LoggerInterface) *API {
	a := new(API)

	a.service = service
	a.tracer = tracer
	a.monitor = monitor
	a.logger = logger

	return a
}
