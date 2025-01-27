package status

import (
	"encoding/json"
	"net/http"

	"github.com/canonical/identity-platform-login-ui/internal/logging"
	"github.com/canonical/identity-platform-login-ui/internal/monitoring"
	"github.com/canonical/identity-platform-login-ui/internal/tracing"
	"github.com/go-chi/chi/v5"
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

type DeploymentInfo struct {
	OidcSequencingEnabled bool `json:"oidc_webauthn_sequencing_enabled"`
}

type API struct {
	oidcWebAuthnSequencingEnabled bool
	service                       ServiceInterface

	tracer tracing.TracingInterface

	monitor monitoring.MonitorInterface
	logger  logging.LoggerInterface
}

func (a *API) RegisterEndpoints(mux *chi.Mux) {
	mux.Get("/api/v0/status", a.alive)
	mux.Get("/api/v0/version", a.version)
	mux.Get("/api/v0/ready", a.ready)
	mux.Get("/api/v0/app-config", a.appConfig)
}

func (a *API) alive(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	rr := Status{
		Status: okValue,
	}

	if buildInfo := a.service.BuildInfo(r.Context()); buildInfo != nil {
		rr.BuildInfo = buildInfo
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(rr)
}

func (a *API) version(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	info := new(BuildInfo)
	if buildInfo := a.service.BuildInfo(r.Context()); buildInfo != nil {
		info = buildInfo
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(info)

}

func (a *API) ready(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	health := new(Health)
	health.Hydra = a.service.HydraStatus(r.Context())
	health.Kratos = a.service.KratosStatus(r.Context())

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(health)
}

func (a *API) appConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	info := new(DeploymentInfo)
	info.OidcSequencingEnabled = a.oidcWebAuthnSequencingEnabled

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(info)
}

func NewAPI(oidcWebAuthnSequencingEnabled bool, service ServiceInterface, tracer tracing.TracingInterface, monitor monitoring.MonitorInterface, logger logging.LoggerInterface) *API {
	a := new(API)

	a.oidcWebAuthnSequencingEnabled = oidcWebAuthnSequencingEnabled
	a.service = service
	a.tracer = tracer
	a.monitor = monitor
	a.logger = logger

	return a
}
