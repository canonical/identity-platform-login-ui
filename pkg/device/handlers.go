// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package device

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/canonical/identity-platform-login-ui/internal/logging"
	"github.com/canonical/identity-platform-login-ui/internal/tracing"
	"github.com/go-chi/chi/v5"
	hClient "github.com/ory/hydra-client-go/v26"
)

type API struct {
	service ServiceInterface

	tracer tracing.TracingInterface
	logger logging.LoggerInterface
}

func (a *API) RegisterEndpoints(mux *chi.Mux) {
	mux.Put("/api/hydra/admin/oauth2/auth/requests/device/accept", a.handleDevice)
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
		var apiErr *hClient.GenericOpenAPIError
		if errors.As(err, &apiErr) && len(apiErr.Body()) > 0 {
			statusCode := http.StatusBadRequest
			var oauth2Err hClient.ErrorOAuth2
			if jsonErr := json.Unmarshal(apiErr.Body(), &oauth2Err); jsonErr == nil && oauth2Err.StatusCode != nil {
				statusCode = int(*oauth2Err.StatusCode)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(statusCode)
			w.Write(apiErr.Body())
			return
		}
		http.Error(w, "Failed to accept user code", http.StatusInternalServerError)
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
