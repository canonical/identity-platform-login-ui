// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package byosso

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/codes"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	tenant "github.com/canonical/identity-platform-api/v0/tenant"

	"github.com/canonical/identity-platform-login-ui/internal/logging"
	"github.com/canonical/identity-platform-login-ui/internal/tracing"
)

// Enforcement is how a tenant enforces its company sign-in. It is off for a
// tenant with no active binding.
type Enforcement string

const (
	EnforcementOff      Enforcement = "off"
	EnforcementOptional Enforcement = "optional"
	EnforcementRequired Enforcement = "required"
)

// MFARequirement is the MFA policy of a tenant.
type MFARequirement string

const (
	MFARequirementNone     MFARequirement = "none"
	MFARequirementRequired MFARequirement = "required"
)

// SignInTenant is one tenant an address may sign in to.
type SignInTenant struct {
	ID                string
	Name              string
	AutoJoinCandidate bool
	// Invited marks a pending invitation of the address, which may have no
	// account yet: its first company sign-in there creates the account.
	Invited bool
}

// SignInContext is what the sign-in screen and the accept checks need to
// know about one tenant and one address or account.
type SignInContext struct {
	Member      bool
	Enforcement Enforcement
	// ConnectionIDs are the active bindings that apply to the address.
	ConnectionIDs []string
	// AutoJoinAdmits and InvitationAdmits are set for an address that is not
	// a member, and may have no account yet. Only a tenant that requires
	// company sign-in has auto-join.
	AutoJoinAdmits   bool
	InvitationAdmits bool
	MFARequirement   MFARequirement
	AccountExists    bool
}

// Admitted reports whether a pending invitation or auto-join admits an
// address that is not a member. Its session is checked as the session of a
// member, and it joins the tenant when the login is accepted.
func (c *SignInContext) Admitted() bool {
	return c != nil && !c.Member && (c.AutoJoinAdmits || c.InvitationAdmits)
}

// Offered reports whether the tenant is one the address may sign in to: a
// member, or admitted by a pending invitation or auto-join.
func (c *SignInContext) Offered() bool {
	return c != nil && (c.Member || c.Admitted())
}

// CompanyOnly reports whether the tenant offers only company sign-ins.
func (c *SignInContext) CompanyOnly() bool {
	return c != nil && c.Enforcement == EnforcementRequired
}

// Applies reports whether connectionID is an active binding that applies.
func (c *SignInContext) Applies(connectionID string) bool {
	if c == nil || connectionID == "" {
		return false
	}
	for _, id := range c.ConnectionIDs {
		if id == connectionID {
			return true
		}
	}
	return false
}

var (
	// errHasTenant is returned when the account belongs to a tenant, or its
	// address has pending invitations, so no personal tenant was created.
	errHasTenant = errors.New("the account belongs to a tenant")
	// errNotAdmitted is returned when the tenant admits the address neither
	// by a pending invitation nor by auto-join.
	errNotAdmitted = errors.New("not admitted")
)

// tenantServiceRetried are the tenant-service calls retried on UNAVAILABLE:
// the reads, the tenant lookup of the handlers on the same channel included,
// and the two writes whose answer is the same a second time.
var tenantServiceRetried = []string{
	tenant.TenantService_LookupTenants_FullMethodName,
	tenant.TenantSignInService_ListSignInTenants_FullMethodName,
	tenant.TenantSignInService_GetSignInContext_FullMethodName,
	tenant.TenantSignInService_JoinTenant_FullMethodName,
	tenant.TenantSignInService_CreatePersonalTenant_FullMethodName,
}

// TenantServiceDialOptions returns the dial options of the channel to
// tenant-service.
func TenantServiceDialOptions(tokens oauth2.TokenSource) []grpc.DialOption {
	return dialOptions(tokens, tenantServiceRetried)
}

// Directory calls tenant-service.
type Directory struct {
	client  TenantSignInServiceClientInterface
	timeout time.Duration

	tracer tracing.TracingInterface
	logger logging.LoggerInterface
}

func (d *Directory) SignInTenants(ctx context.Context, email string) ([]SignInTenant, error) {
	ctx, span := d.tracer.Start(ctx, "byosso.Directory.SignInTenants")
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()

	resp, err := d.client.ListSignInTenants(ctx, &tenant.ListSignInTenantsRequest{Email: email})
	if err != nil {
		d.logger.Debugf("full gRPC status: %v", status.Convert(err).Proto())
		span.RecordError(err)
		span.SetStatus(codes.Error, "cannot list the sign-in tenants of an address")
		return nil, fmt.Errorf("cannot list the sign-in tenants of an address: %w", err)
	}

	tenants := make([]SignInTenant, 0, len(resp.GetTenants()))
	for _, t := range resp.GetTenants() {
		if t.GetTenant() == nil {
			continue
		}
		tenants = append(tenants, SignInTenant{
			ID:                t.GetTenant().GetId(),
			Name:              t.GetTenant().GetName(),
			AutoJoinCandidate: t.GetAutoJoinCandidate(),
			Invited:           t.GetInvited(),
		})
	}

	span.SetStatus(codes.Ok, "")
	return tenants, nil
}

func (d *Directory) SignInContext(ctx context.Context, tenantID, email, identityID string) (*SignInContext, error) {
	ctx, span := d.tracer.Start(ctx, "byosso.Directory.SignInContext")
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()

	resp, err := d.client.GetSignInContext(ctx, &tenant.GetSignInContextRequest{TenantId: tenantID, Email: email, IdentityId: identityID})
	if err != nil {
		d.logger.Debugf("full gRPC status: %v", status.Convert(err).Proto())
		span.RecordError(err)
		span.SetStatus(codes.Error, "cannot get the sign-in context")
		return nil, fmt.Errorf("cannot get the sign-in context of tenant %s: %w", tenantID, err)
	}

	sc, err := toSignInContext(resp.GetContext())
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return sc, nil
}

// JoinTenant makes the account identityID a member of tenantID. It returns
// errNotAdmitted when the tenant admits its address neither by a pending
// invitation nor by auto-join.
func (d *Directory) JoinTenant(ctx context.Context, tenantID, identityID string) error {
	ctx, span := d.tracer.Start(ctx, "byosso.Directory.JoinTenant")
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()

	if _, err := d.client.JoinTenant(ctx, &tenant.JoinTenantRequest{TenantId: tenantID, IdentityId: identityID}); err != nil {
		d.logger.Debugf("full gRPC status: %v", status.Convert(err).Proto())
		span.RecordError(err)
		span.SetStatus(codes.Error, "cannot join the tenant")
		if st, ok := status.FromError(err); ok && errorReason(st) == reasonNotAdmitted {
			return fmt.Errorf("cannot join tenant %s: %w", tenantID, errNotAdmitted)
		}
		return fmt.Errorf("cannot join tenant %s: %w", tenantID, err)
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// CreatePersonalTenant returns the personal tenant of the account, creating
// it if absent. It returns errHasTenant when the account already belongs to
// a tenant or its address has pending invitations, which wait for a sign-in
// to their tenant.
func (d *Directory) CreatePersonalTenant(ctx context.Context, identityID string) (string, error) {
	ctx, span := d.tracer.Start(ctx, "byosso.Directory.CreatePersonalTenant")
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()

	resp, err := d.client.CreatePersonalTenant(ctx, &tenant.CreatePersonalTenantRequest{IdentityId: identityID})
	if err != nil {
		if st, ok := status.FromError(err); ok && errorReason(st) == reasonHasTenant {
			span.SetStatus(codes.Ok, "")
			return "", errHasTenant
		}
		d.logger.Debugf("full gRPC status: %v", status.Convert(err).Proto())
		span.RecordError(err)
		span.SetStatus(codes.Error, "cannot create the personal tenant")
		return "", fmt.Errorf("cannot create the personal tenant of %s: %w", identityID, err)
	}

	id := resp.GetTenant().GetId()
	if id == "" {
		err := fmt.Errorf("empty tenant from tenant service")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}

	span.SetStatus(codes.Ok, "")
	return id, nil
}

// toSignInContext converts the proto. An unspecified or unknown enum value is
// an error, so that the caller fails closed rather than read it as the
// weakest value.
func toSignInContext(c *tenant.SignInContext) (*SignInContext, error) {
	if c == nil {
		return nil, fmt.Errorf("empty sign-in context from tenant service")
	}

	sc := &SignInContext{
		Member:           c.GetMember(),
		ConnectionIDs:    c.GetConnectionIds(),
		AutoJoinAdmits:   c.GetAutoJoinAdmits(),
		InvitationAdmits: c.GetInvitationAdmits(),
		AccountExists:    c.GetAccountExists(),
	}

	switch c.GetEnforcement() {
	case tenant.Enforcement_ENFORCEMENT_OFF:
		sc.Enforcement = EnforcementOff
		// A tenant with no active binding offers no company sign-in.
		sc.ConnectionIDs = nil
	case tenant.Enforcement_ENFORCEMENT_OPTIONAL:
		sc.Enforcement = EnforcementOptional
	case tenant.Enforcement_ENFORCEMENT_REQUIRED:
		sc.Enforcement = EnforcementRequired
	default:
		return nil, fmt.Errorf("unknown enforcement %v", c.GetEnforcement())
	}

	switch c.GetMfaRequirement() {
	case tenant.MFARequirement_MFA_REQUIREMENT_NONE:
		sc.MFARequirement = MFARequirementNone
	case tenant.MFARequirement_MFA_REQUIREMENT_REQUIRED:
		sc.MFARequirement = MFARequirementRequired
	default:
		return nil, fmt.Errorf("unknown MFA requirement %v", c.GetMfaRequirement())
	}

	return sc, nil
}

func NewDirectory(client TenantSignInServiceClientInterface, timeout time.Duration, tracer tracing.TracingInterface, logger logging.LoggerInterface) *Directory {
	d := new(Directory)

	if timeout <= 0 {
		timeout = defaultGRPCTimeout
	}

	d.client = client
	d.timeout = timeout

	d.tracer = tracer
	d.logger = logger

	return d
}
