// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package tenants

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/codes"

	tenant "github.com/canonical/identity-platform-api/v0/tenant"
	kClient "github.com/ory/kratos-client-go/v25"

	"github.com/canonical/identity-platform-login-ui/internal/logging"
	"github.com/canonical/identity-platform-login-ui/internal/monitoring"
	"github.com/canonical/identity-platform-login-ui/internal/tracing"
)

// Tenant represents a tenant entry returned by the external tenants API.
type Tenant struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
	Enabled   bool   `json:"enabled"`
}

// Service fetches tenant data from the tenant gRPC service.
type Service struct {
	grpcClient  TenantServiceClientInterface
	flowFetcher FlowFetcherInterface
	timeout     time.Duration

	tracer  tracing.TracingInterface
	monitor monitoring.MonitorInterface
	logger  logging.LoggerInterface
}

func (s *Service) lookupTenantsByEmail(ctx context.Context, email string) ([]*Tenant, error) {
	ctx, span := s.tracer.Start(ctx, "tenants.Service.lookupTenantsByEmail")
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	resp, err := s.grpcClient.LookupTenants(ctx, &tenant.LookupTenantsRequest{Email: email})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "cannot lookup tenants by email")
		return nil, fmt.Errorf("cannot lookup tenants by email: %w", err)
	}

	if resp == nil {
		err := fmt.Errorf("empty response from tenant service")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return toLocalTenants(resp.Tenants), nil
}

// LookupTenantsByEmail looks up tenants for the given email address directly,
// without requiring a Kratos login flow. This is used when the user already
// has an active Kratos session and a flow has not yet been created.
func (s *Service) LookupTenantsByEmail(ctx context.Context, email string) ([]*Tenant, error) {
	return s.lookupTenantsByEmail(ctx, email)
}

// LookupTenantsByIdentityID looks up tenants for the given Kratos identity ID.
// This skips the Kratos email-to-identity resolution on the tenant-service side,
// resulting in faster lookups. Used when the caller already knows the identity ID
// (e.g., from an active Kratos session).
func (s *Service) LookupTenantsByIdentityID(ctx context.Context, identityID string) ([]*Tenant, error) {
	ctx, span := s.tracer.Start(ctx, "tenants.Service.LookupTenantsByIdentityID")
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	resp, err := s.grpcClient.LookupTenants(ctx, &tenant.LookupTenantsRequest{IdentityId: identityID})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "cannot lookup tenants by identity id")
		return nil, fmt.Errorf("cannot lookup tenants by identity id: %w", err)
	}

	if resp == nil {
		err := fmt.Errorf("empty response from tenant service")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return toLocalTenants(resp.Tenants), nil
}

// LookupTenantsByFlow fetches the Kratos login flow, extracts the email from
// the flow's UI nodes, and looks up tenants for that email.
func (s *Service) LookupTenantsByFlow(ctx context.Context, flowID string, cookies []*http.Cookie) ([]*Tenant, error) {
	ctx, span := s.tracer.Start(ctx, "tenants.Service.LookupTenantsByFlow")
	defer span.End()

	flow, _, err := s.flowFetcher.GetLoginFlow(ctx, flowID, cookies)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fetch login flow")
		return nil, fmt.Errorf("failed to fetch login flow: %w", err)
	}

	email, err := emailFromFlow(flow)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to extract email from flow")
		return nil, err
	}

	return s.lookupTenantsByEmail(ctx, email)
}

// emailFromFlow extracts the email from a login flow's "identifier" UI node.
func emailFromFlow(flow *kClient.LoginFlow) (string, error) {
	for _, node := range flow.Ui.Nodes {
		if node.Type == "input" &&
			node.Attributes.UiNodeInputAttributes != nil &&
			node.Attributes.UiNodeInputAttributes.Name == "identifier" {
			val, ok := node.Attributes.UiNodeInputAttributes.Value.(string)
			if !ok || val == "" {
				return "", fmt.Errorf("identifier value is not a string or is empty in flow %s", flow.Id)
			}
			return val, nil
		}
	}
	return "", fmt.Errorf("identifier node not found in flow %s", flow.Id)
}

// toLocalTenants converts a slice of gRPC Tenant protos to local Tenant structs.
// Nil entries in the slice are silently skipped to guard against malformed upstream data.
func toLocalTenants(ts []*tenant.Tenant) []*Tenant {
	result := make([]*Tenant, 0, len(ts))
	for _, t := range ts {
		if t == nil {
			continue
		}
		result = append(result, &Tenant{
			ID:        t.Id,
			Name:      t.Name,
			CreatedAt: t.CreatedAt,
			Enabled:   t.Enabled,
		})
	}
	return result
}

const defaultGRPCTimeout = 5 * time.Second

func NewService(grpcClient TenantServiceClientInterface, flowFetcher FlowFetcherInterface, timeout time.Duration, tracer tracing.TracingInterface, monitor monitoring.MonitorInterface, logger logging.LoggerInterface) *Service {
	if timeout <= 0 {
		timeout = defaultGRPCTimeout
	}
	return &Service{
		grpcClient:  grpcClient,
		flowFetcher: flowFetcher,
		timeout:     timeout,
		tracer:      tracer,
		monitor:     monitor,
		logger:      logger,
	}
}
