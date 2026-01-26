package device

import (
	"encoding/json"
	"net/http"

	"github.com/canonical/identity-platform-login-ui/internal/logging"
	"github.com/canonical/identity-platform-login-ui/internal/tracing"
	"github.com/go-chi/chi/v5"
)

type API struct {
	service ServiceInterface

	tracer tracing.TracingInterface
	logger logging.LoggerInterface
}

const NOT_FOUND_ERROR_DESC = "The user_code provided is either invalid, expired or already used."

func (a *API) RegisterEndpoints(mux *chi.Mux) {
	mux.Put("/api/device", a.handleDevice)
}

func (a *API) handleDevice(w http.ResponseWriter, r *http.Request) {
	challenge := r.URL.Query().Get("device_challenge")

	body, err := a.service.ParseUserCodeBody(r)
	if err != nil {
		a.logger.Errorf("Error when parsing request body: %v\n", err)
		http.Error(w, "Failed to parse user code", http.StatusInternalServerError)
		return
	}

	deviceResp, err := a.service.AcceptUserCode(r.Context(), challenge, body)
	if err != nil {
		a.logger.Errorf("Failed to accept user code: %v\n", err)
		if e := err.Error(); e == "404 Not Found" {
			http.Error(w, NOT_FOUND_ERROR_DESC, http.StatusBadRequest)
		} else {
			http.Error(w, "Failed to accept user code", http.StatusInternalServerError)
		}
		return
	}
	resp, err := json.Marshal(deviceResp)
	if err != nil {
		a.logger.Errorf("Error when marshalling Json: %v\n", err)
		http.Error(w, "Failed to parse response", http.StatusInternalServerError)
		return
	}

	w.Write(resp)
	w.WriteHeader(http.StatusOK)
}

func NewAPI(service ServiceInterface, tracer tracing.TracingInterface, logger logging.LoggerInterface) *API {
	a := new(API)

	a.service = service

	a.tracer = tracer
	a.logger = logger

	return a
}
