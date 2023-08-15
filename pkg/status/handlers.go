package status

import (
	"encoding/json"
	"net/http"
	"sync"

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

type DeepCheckStatus struct {
	KratosStatus string `json:"kratos_status"`
	HydraStatus  string `json:"hydra_status"`
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
	mux.Get("/api/v0/deepcheck", a.deepCheck)
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

func (a *API) deepCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	wg := sync.WaitGroup{}
	wg.Add(2)

	kratosOK := false
	hydraOK := false

	go func() {
		res, err := a.service.CheckKratosReady(r.Context())
		if err != nil {
			a.logger.Errorf("error when checking kratos status: %s", err)
		}
		kratosOK = res
		wg.Done()
	}()

	go func() {
		res, err := a.service.CheckHydraReady(r.Context())
		if err != nil {
			a.logger.Errorf("error when checking hydra status: %s", err)
		}
		hydraOK = res
		wg.Done()
	}()

	wg.Wait()

	msgMap := func(ok bool) string {
		if ok {
			return okValue
		}
		return "unavailable"
	}

	ds := DeepCheckStatus{
		KratosStatus: msgMap(kratosOK),
		HydraStatus:  msgMap(hydraOK),
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(ds)
}

func NewAPI(service ServiceInterface, tracer trace.Tracer, monitor monitoring.MonitorInterface, logger logging.LoggerInterface) *API {
	a := new(API)

	a.service = service
	a.tracer = tracer
	a.monitor = monitor
	a.logger = logger

	return a
}
