package tenants

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/canonical/identity-platform-login-ui/internal/logging"
)

// API exposes the tenant selection endpoint.
type API struct {
	service ServiceInterface
	logger  logging.LoggerInterface
}

func (a *API) RegisterEndpoints(mux *chi.Mux) {
	mux.Get("/api/v0/users/{userID}/tenants", a.handleGetUserTenants)
}

func (a *API) handleGetUserTenants(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	if userID == "" {
		a.logger.Errorf("userID is required")
		http.Error(w, "userID is required", http.StatusBadRequest)
		return
	}

	tenants, err := a.service.GetUserTenants(r.Context(), userID)
	if err != nil {
		a.logger.Errorf("failed to get tenants for user %s: %v", userID, err)
		http.Error(w, "failed to get tenants", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(tenants)
}

func NewAPI(service ServiceInterface, logger logging.LoggerInterface) *API {
	return &API{
		service: service,
		logger:  logger,
	}
}
