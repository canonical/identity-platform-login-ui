// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package tenants

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	tenant "github.com/canonical/identity-platform-api/v0/tenant"
	kClient "github.com/ory/kratos-client-go/v25"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
)

// noopTracer satisfies tracing.TracingInterface without any side effects.
type noopTracer struct{}

func (n *noopTracer) Start(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
	return ctx, trace.SpanFromContext(ctx)
}

const testTimeout = 5 * time.Second

func newServiceForTest(grpcClient TenantServiceClientInterface, flowFetcher FlowFetcherInterface) *Service {
	return NewService(grpcClient, flowFetcher, testTimeout, &noopTracer{}, nil, nil)
}

// protoTenants builds a slice of proto Tenant values for use in test responses.
func protoTenants(ids ...string) []*tenant.Tenant {
	ts := make([]*tenant.Tenant, 0, len(ids))
	for _, id := range ids {
		ts = append(ts, &tenant.Tenant{Id: id, Name: "name-" + id, CreatedAt: "2026-01-01", Enabled: true})
	}
	return ts
}

func TestLookupTenantsByEmailSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockGRPC := NewMockTenantServiceClientInterface(ctrl)
	svc := newServiceForTest(mockGRPC, nil)

	mockGRPC.EXPECT().
		LookupTenants(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, req *tenant.LookupTenantsRequest, _ ...grpc.CallOption) (*tenant.LookupTenantsResponse, error) {
			if req.Email != "user@example.com" {
				t.Errorf("expected email %q, got %q", "user@example.com", req.Email)
			}
			return &tenant.LookupTenantsResponse{Tenants: protoTenants("t1", "t2")}, nil
		})

	got, err := svc.LookupTenantsByEmail(ctx, "user@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 tenants, got %d", len(got))
	}
	if got[0].ID != "t1" || got[1].ID != "t2" {
		t.Fatalf("unexpected tenant IDs: %v, %v", got[0].ID, got[1].ID)
	}
}

func TestLookupTenantsByEmailGRPCError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockGRPC := NewMockTenantServiceClientInterface(ctrl)
	svc := newServiceForTest(mockGRPC, nil)

	mockGRPC.EXPECT().LookupTenants(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("rpc error"))

	_, err := svc.LookupTenantsByEmail(ctx, "user@example.com")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLookupTenantsByEmailNilResponse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockGRPC := NewMockTenantServiceClientInterface(ctrl)
	svc := newServiceForTest(mockGRPC, nil)

	mockGRPC.EXPECT().LookupTenants(gomock.Any(), gomock.Any()).Return(nil, nil)

	_, err := svc.LookupTenantsByEmail(ctx, "user@example.com")
	if err == nil {
		t.Fatal("expected error for nil response, got nil")
	}
}

func TestLookupTenantsByIdentityIDSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockGRPC := NewMockTenantServiceClientInterface(ctrl)
	svc := newServiceForTest(mockGRPC, nil)

	mockGRPC.EXPECT().
		LookupTenants(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, req *tenant.LookupTenantsRequest, _ ...grpc.CallOption) (*tenant.LookupTenantsResponse, error) {
			if req.IdentityId != "identity-abc" {
				t.Errorf("expected identity_id %q, got %q", "identity-abc", req.IdentityId)
			}
			return &tenant.LookupTenantsResponse{Tenants: protoTenants("t3")}, nil
		})

	got, err := svc.LookupTenantsByIdentityID(ctx, "identity-abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].ID != "t3" {
		t.Fatalf("unexpected result: %v", got)
	}
}

func TestLookupTenantsByIdentityIDGRPCError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockGRPC := NewMockTenantServiceClientInterface(ctrl)
	svc := newServiceForTest(mockGRPC, nil)

	mockGRPC.EXPECT().LookupTenants(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("unavailable"))

	_, err := svc.LookupTenantsByIdentityID(ctx, "identity-abc")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLookupTenantsByIdentityIDNilResponse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockGRPC := NewMockTenantServiceClientInterface(ctrl)
	svc := newServiceForTest(mockGRPC, nil)

	mockGRPC.EXPECT().LookupTenants(gomock.Any(), gomock.Any()).Return(nil, nil)

	_, err := svc.LookupTenantsByIdentityID(ctx, "identity-abc")
	if err == nil {
		t.Fatal("expected error for nil response, got nil")
	}
}

func TestLookupTenantsByFlowSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockGRPC := NewMockTenantServiceClientInterface(ctrl)
	mockFetcher := NewMockFlowFetcherInterface(ctrl)
	svc := newServiceForTest(mockGRPC, mockFetcher)

	flow := buildFlowWithIdentifier("user@example.com")

	mockFetcher.EXPECT().GetLoginFlow(ctx, "flow-1", []*http.Cookie{}).Return(flow, nil, nil)
	mockGRPC.EXPECT().
		LookupTenants(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, req *tenant.LookupTenantsRequest, _ ...grpc.CallOption) (*tenant.LookupTenantsResponse, error) {
			if req.Email != "user@example.com" {
				t.Errorf("expected email %q, got %q", "user@example.com", req.Email)
			}
			return &tenant.LookupTenantsResponse{Tenants: protoTenants("t1")}, nil
		})

	got, err := svc.LookupTenantsByFlow(ctx, "flow-1", []*http.Cookie{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].ID != "t1" {
		t.Fatalf("unexpected result: %v", got)
	}
}

func TestLookupTenantsByFlowFetchError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockFetcher := NewMockFlowFetcherInterface(ctrl)
	svc := newServiceForTest(nil, mockFetcher)

	mockFetcher.EXPECT().GetLoginFlow(ctx, "flow-1", []*http.Cookie{}).Return(nil, nil, fmt.Errorf("kratos error"))

	_, err := svc.LookupTenantsByFlow(ctx, "flow-1", []*http.Cookie{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLookupTenantsByFlowMissingIdentifier(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockFetcher := NewMockFlowFetcherInterface(ctrl)
	svc := newServiceForTest(nil, mockFetcher)

	// Flow with no identifier node.
	flow := kClient.NewLoginFlowWithDefaults()
	flow.Id = "flow-2"
	flow.Ui = kClient.UiContainer{Nodes: []kClient.UiNode{}}

	mockFetcher.EXPECT().GetLoginFlow(ctx, "flow-2", []*http.Cookie{}).Return(flow, nil, nil)

	_, err := svc.LookupTenantsByFlow(ctx, "flow-2", []*http.Cookie{})
	if err == nil {
		t.Fatal("expected error for missing identifier, got nil")
	}
}

func TestToLocalTenants(t *testing.T) {
	tests := []struct {
		name  string
		input []*tenant.Tenant
		want  []*Tenant
	}{
		{
			name:  "nil input",
			input: nil,
			want:  []*Tenant{},
		},
		{
			name:  "empty input",
			input: []*tenant.Tenant{},
			want:  []*Tenant{},
		},
		{
			name: "single tenant",
			input: []*tenant.Tenant{
				{Id: "t1", Name: "Tenant One", CreatedAt: "2026-01-01", Enabled: true},
			},
			want: []*Tenant{
				{ID: "t1", Name: "Tenant One", CreatedAt: "2026-01-01", Enabled: true},
			},
		},
		{
			name: "multiple tenants preserves order",
			input: []*tenant.Tenant{
				{Id: "a", Name: "Alpha", Enabled: true},
				{Id: "b", Name: "Beta", Enabled: false},
			},
			want: []*Tenant{
				{ID: "a", Name: "Alpha", Enabled: true},
				{ID: "b", Name: "Beta", Enabled: false},
			},
		},
		{
			name: "nil entry is skipped",
			input: []*tenant.Tenant{
				{Id: "x", Name: "X", Enabled: true},
				nil,
				{Id: "y", Name: "Y", Enabled: false},
			},
			want: []*Tenant{
				{ID: "x", Name: "X", Enabled: true},
				{ID: "y", Name: "Y", Enabled: false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toLocalTenants(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("expected %d tenants, got %d", len(tt.want), len(got))
			}
			for i, w := range tt.want {
				g := got[i]
				if g.ID != w.ID || g.Name != w.Name || g.CreatedAt != w.CreatedAt || g.Enabled != w.Enabled {
					t.Errorf("tenant[%d]: expected %+v, got %+v", i, w, g)
				}
			}
		})
	}
}

func TestEmailFromFlow(t *testing.T) {
	tests := []struct {
		name      string
		flow      *kClient.LoginFlow
		wantEmail string
		wantErr   bool
	}{
		{
			name:      "identifier node present",
			flow:      buildFlowWithIdentifier("user@example.com"),
			wantEmail: "user@example.com",
		},
		{
			name: "no identifier node",
			flow: func() *kClient.LoginFlow {
				f := kClient.NewLoginFlowWithDefaults()
				f.Id = "f1"
				f.Ui = kClient.UiContainer{Nodes: []kClient.UiNode{}}
				return f
			}(),
			wantErr: true,
		},
		{
			name:    "identifier node with empty value",
			flow:    buildFlowWithIdentifier(""),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			email, err := emailFromFlow(tt.flow)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if email != tt.wantEmail {
				t.Fatalf("expected %q, got %q", tt.wantEmail, email)
			}
		})
	}
}

// buildFlowWithIdentifier constructs a minimal LoginFlow whose UI has an
// "identifier" input node pre-filled with the given email.
func buildFlowWithIdentifier(email string) *kClient.LoginFlow {
	node := kClient.UiNode{
		Type:  "input",
		Group: "default",
		Attributes: kClient.UiNodeAttributes{
			UiNodeInputAttributes: &kClient.UiNodeInputAttributes{
				Name:  "identifier",
				Value: email,
				Type:  "email",
			},
		},
		Messages: []kClient.UiText{},
		Meta:     kClient.UiNodeMeta{},
	}
	flow := kClient.NewLoginFlowWithDefaults()
	flow.Id = "flow-1"
	flow.Ui = kClient.UiContainer{Nodes: []kClient.UiNode{node}}
	return flow
}

func TestNewServiceZeroTimeoutDefaulted(t *testing.T) {
	svc := NewService(nil, nil, 0, &noopTracer{}, nil, nil)
	if svc.timeout <= 0 {
		t.Fatalf("expected timeout to be defaulted to a positive value, got %v", svc.timeout)
	}
}

func TestNewServiceNegativeTimeoutDefaulted(t *testing.T) {
	svc := NewService(nil, nil, -1*time.Second, &noopTracer{}, nil, nil)
	if svc.timeout <= 0 {
		t.Fatalf("expected timeout to be defaulted to a positive value, got %v", svc.timeout)
	}
}
