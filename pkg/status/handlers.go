package status

import (
	"encoding/json"
	"net/http"
)

const okValue = "ok"

type Status struct {
	Status    string     `json:"status"`
	BuildInfo *BuildInfo `json:"buildInfo"`
}

type API struct{}

func (a *API) RegisterEndpoints(mux *http.ServeMux) {
	mux.HandleFunc("/health/alive", a.handleAlive)
	mux.HandleFunc("/health/version", a.version)

}

func (a *API) handleAlive(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	rr := Status{
		Status: okValue,
	}

	if buildInfo := buildInfo(); buildInfo != nil {
		rr.BuildInfo = buildInfo
	}

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

func NewAPI() *API {
	return new(API)
}
