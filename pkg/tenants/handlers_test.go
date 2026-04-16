// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0

package tenants

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	kClient "github.com/ory/kratos-client-go/v25"
	"go.uber.org/mock/gomock"
)

//go:generate mockgen -build_flags=--mod=mod -package tenants -destination ./mock_logger.go -source=../../internal/logging/interfaces.go
//go:generate mockgen -build_flags=--mod=mod -package tenants -destination ./mock_tracing.go -source=../../internal/tracing/interfaces.go
//go:generate mockgen -build_flags=--mod=mod -package tenants -destination ./mock_interfaces.go -source=./interfaces.go
//go:generate mockgen -build_flags=--mod=mod -package tenants -destination ./mock_monitor.go -source=../../internal/monitoring/interfaces.go

func TestHandleLookupTenantsBySessionUnauthorized(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)
	mockSessionChecker := NewMockSessionCheckerInterface(ctrl)

	mockSessionChecker.EXPECT().CheckSession(gomock.Any(), gomock.Any()).Return(nil, nil, errors.New("no session"))

	mux := chi.NewMux()
	NewAPI(mockService, nil, mockSessionChecker, "", mockTracer, mockLogger).RegisterEndpoints(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v0/tenants", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestHandleLookupTenantsBySessionSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)
	mockSessionChecker := NewMockSessionCheckerInterface(ctrl)

	identityID := "identity-123"
	session := &kClient.Session{
		Identity: &kClient.Identity{
			Id:     identityID,
			Traits: map[string]interface{}{"email": "user@example.com"},
		},
	}
	expected := []Tenant{{ID: "t1", Name: "Acme", Enabled: true}}

	mockSessionChecker.EXPECT().CheckSession(gomock.Any(), gomock.Any()).Return(session, nil, nil)
	mockService.EXPECT().LookupTenantsByIdentityID(gomock.Any(), identityID).Return(expected, nil)

	mux := chi.NewMux()
	NewAPI(mockService, nil, mockSessionChecker, "", mockTracer, mockLogger).RegisterEndpoints(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v0/tenants", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestHandleLookupTenantsSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)

	flowID := "flow-123"
	expected := []Tenant{{ID: "t1", Name: "Acme", Enabled: true}}

	mockService.EXPECT().LookupTenantsByFlow(gomock.Any(), flowID, gomock.Any()).Return(expected, nil)

	mux := chi.NewMux()
	NewAPI(mockService, nil, nil, "", mockTracer, mockLogger).RegisterEndpoints(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v0/tenants?flow="+flowID, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestHandleLookupTenantsServiceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)

	flowID := "flow-123"

	mockService.EXPECT().LookupTenantsByFlow(gomock.Any(), flowID, gomock.Any()).Return(nil, errors.New("upstream error"))
	mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any())

	mux := chi.NewMux()
	NewAPI(mockService, nil, nil, "", mockTracer, mockLogger).RegisterEndpoints(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v0/tenants?flow="+flowID, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestHandleTenantSelectionRejectsSentinel(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)

	mux := chi.NewMux()
	NewAPI(nil, nil, nil, "", mockTracer, mockLogger).RegisterEndpoints(mux)

	body, _ := json.Marshal(tenantSelectionRequest{
		LoginChallenge: "lc-1",
		TenantID:       "_none",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v0/auth/tenant", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleTenantSelectionEmptyVerifiedNoTenants(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)
	mockSessionChecker := NewMockSessionCheckerInterface(ctrl)
	mockStorer := NewMockTenantStorerInterface(ctrl)

	identityID := "identity-123"
	session := &kClient.Session{
		Identity: &kClient.Identity{
			Id:     identityID,
			Traits: map[string]interface{}{"email": "user@example.com"},
		},
	}

	// No flow provided → falls back to session lookup.
	mockSessionChecker.EXPECT().CheckSession(gomock.Any(), gomock.Any()).Return(session, nil, nil)
	mockService.EXPECT().LookupTenantsByIdentityID(gomock.Any(), identityID).Return([]Tenant{}, nil)
	mockStorer.EXPECT().StoreTenant(gomock.Any(), gomock.Any(), "_none", "lc-1").Return(nil)

	mux := chi.NewMux()
	NewAPI(mockService, mockStorer, mockSessionChecker, "http://localhost", mockTracer, mockLogger).RegisterEndpoints(mux)

	body, _ := json.Marshal(tenantSelectionRequest{LoginChallenge: "lc-1"})
	req := httptest.NewRequest(http.MethodPost, "/api/v0/auth/tenant", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestHandleTenantSelectionEmptyRejectedWhenTenantsExist(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)
	mockSessionChecker := NewMockSessionCheckerInterface(ctrl)

	identityID := "identity-123"
	session := &kClient.Session{
		Identity: &kClient.Identity{
			Id:     identityID,
			Traits: map[string]interface{}{"email": "user@example.com"},
		},
	}

	mockSessionChecker.EXPECT().CheckSession(gomock.Any(), gomock.Any()).Return(session, nil, nil)
	mockService.EXPECT().LookupTenantsByIdentityID(gomock.Any(), identityID).Return([]Tenant{{ID: "t1", Name: "Acme", Enabled: true}}, nil)

	mux := chi.NewMux()
	NewAPI(mockService, nil, mockSessionChecker, "", mockTracer, mockLogger).RegisterEndpoints(mux)

	body, _ := json.Marshal(tenantSelectionRequest{LoginChallenge: "lc-1"})
	req := httptest.NewRequest(http.MethodPost, "/api/v0/auth/tenant", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
