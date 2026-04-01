// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0

package tenants

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	kClient "github.com/ory/kratos-client-go/v25"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/mock/gomock"
)

func TestResolveTenantZeroTenants(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockMonitor := NewMockMonitorInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockFlowFetcher := NewMockFlowFetcherInterface(ctrl)

	ctx := context.Background()

	// Tenants API returns empty list
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(struct {
			Tenants []Tenant `json:"tenants"`
		}{})
	}))
	defer srv.Close()

	mockTracer.EXPECT().Start(gomock.Any(), "tenants.Service.ResolveTenant").Return(ctx, trace.SpanFromContext(ctx))
	mockTracer.EXPECT().Start(gomock.Any(), "tenants.Service.LookupTenantsByFlow").Return(ctx, trace.SpanFromContext(ctx))
	mockTracer.EXPECT().Start(gomock.Any(), "tenants.Service.lookupTenantsByEmail").Return(ctx, trace.SpanFromContext(ctx))

	mockFlowFetcher.EXPECT().GetLoginFlow(gomock.Any(), "flow-1", gomock.Any()).Return(
		makeFlowWithIdentifier("flow-1", "user@example.com"), []*http.Cookie{}, nil,
	)

	svc := NewService(srv.URL, "", mockFlowFetcher, mockTracer, mockMonitor, mockLogger)
	result, err := svc.ResolveTenant(ctx, "flow-1", "lc-1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.RedirectTo != "" {
		t.Fatalf("expected empty redirect, got %q", result.RedirectTo)
	}
}

func TestResolveTenantSingleTenant(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockMonitor := NewMockMonitorInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockFlowFetcher := NewMockFlowFetcherInterface(ctrl)

	ctx := context.Background()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(struct {
			Tenants []Tenant `json:"tenants"`
		}{Tenants: []Tenant{{ID: "t1", Name: "Acme", Enabled: true}}})
	}))
	defer srv.Close()

	mockTracer.EXPECT().Start(gomock.Any(), "tenants.Service.ResolveTenant").Return(ctx, trace.SpanFromContext(ctx))
	mockTracer.EXPECT().Start(gomock.Any(), "tenants.Service.LookupTenantsByFlow").Return(ctx, trace.SpanFromContext(ctx))
	mockTracer.EXPECT().Start(gomock.Any(), "tenants.Service.lookupTenantsByEmail").Return(ctx, trace.SpanFromContext(ctx))

	mockFlowFetcher.EXPECT().GetLoginFlow(gomock.Any(), "flow-1", gomock.Any()).Return(
		makeFlowWithIdentifier("flow-1", "user@example.com"), []*http.Cookie{}, nil,
	)

	svc := NewService(srv.URL, "", mockFlowFetcher, mockTracer, mockMonitor, mockLogger)
	result, err := svc.ResolveTenant(ctx, "flow-1", "lc-1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Single-tenant: service must return the tenant ID for the handler to store.
	// RedirectTo is intentionally empty here; the handler builds the login URL.
	if result.TenantID == "" {
		t.Fatal("expected TenantID to be set for single tenant")
	}
	if result.TenantID != "t1" {
		t.Fatalf("expected TenantID=t1, got %q", result.TenantID)
	}
	if result.RedirectTo != "" {
		t.Fatalf("expected empty RedirectTo for single tenant, got %q", result.RedirectTo)
	}
}

func TestResolveTenantMultipleTenants(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockMonitor := NewMockMonitorInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockFlowFetcher := NewMockFlowFetcherInterface(ctrl)

	ctx := context.Background()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(struct {
			Tenants []Tenant `json:"tenants"`
		}{Tenants: []Tenant{
			{ID: "t1", Name: "Acme", Enabled: true},
			{ID: "t2", Name: "Globex", Enabled: true},
		}})
	}))
	defer srv.Close()

	mockTracer.EXPECT().Start(gomock.Any(), "tenants.Service.ResolveTenant").Return(ctx, trace.SpanFromContext(ctx))
	mockTracer.EXPECT().Start(gomock.Any(), "tenants.Service.LookupTenantsByFlow").Return(ctx, trace.SpanFromContext(ctx))
	mockTracer.EXPECT().Start(gomock.Any(), "tenants.Service.lookupTenantsByEmail").Return(ctx, trace.SpanFromContext(ctx))

	mockFlowFetcher.EXPECT().GetLoginFlow(gomock.Any(), "flow-1", gomock.Any()).Return(
		makeFlowWithIdentifier("flow-1", "user@example.com"), []*http.Cookie{}, nil,
	)

	svc := NewService(srv.URL, "", mockFlowFetcher, mockTracer, mockMonitor, mockLogger)
	result, err := svc.ResolveTenant(ctx, "flow-1", "lc-1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.RedirectTo == "" {
		t.Fatal("expected redirect for multiple tenants")
	}
	if !strings.Contains(result.RedirectTo, "select_tenant") {
		t.Fatalf("expected select_tenant URL, got %q", result.RedirectTo)
	}
}

func TestResolveTenantLookupError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockMonitor := NewMockMonitorInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockFlowFetcher := NewMockFlowFetcherInterface(ctrl)

	ctx := context.Background()

	mockTracer.EXPECT().Start(gomock.Any(), "tenants.Service.ResolveTenant").Return(ctx, trace.SpanFromContext(ctx))
	mockTracer.EXPECT().Start(gomock.Any(), "tenants.Service.LookupTenantsByFlow").Return(ctx, trace.SpanFromContext(ctx))

	mockFlowFetcher.EXPECT().GetLoginFlow(gomock.Any(), "flow-1", gomock.Any()).Return(
		nil, nil, fmt.Errorf("kratos unavailable"),
	)

	svc := NewService("http://unused", "", mockFlowFetcher, mockTracer, mockMonitor, mockLogger)
	_, err := svc.ResolveTenant(ctx, "flow-1", "lc-1", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// makeFlowWithIdentifier creates a minimal LoginFlow with an identifier node.
func makeFlowWithIdentifier(flowID, email string) *kClient.LoginFlow {
	return &kClient.LoginFlow{
		Id: flowID,
		Ui: kClient.UiContainer{
			Nodes: []kClient.UiNode{
				{
					Type: "input",
					Attributes: kClient.UiNodeAttributes{
						UiNodeInputAttributes: &kClient.UiNodeInputAttributes{
							Name:  "identifier",
							Value: email,
						},
					},
				},
			},
		},
	}
}
