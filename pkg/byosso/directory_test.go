// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package byosso

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	tenant "github.com/canonical/identity-platform-api/v0/tenant"
)

func TestSignInContextAdmitted(t *testing.T) {
	var noContext *SignInContext

	tests := []struct {
		name                string
		context             *SignInContext
		expectedAdmitted    bool
		expectedOffered     bool
		expectedCompanyOnly bool
	}{
		{name: "member", context: member(EnforcementOptional, MFARequirementNone, connA), expectedOffered: true},
		{name: "member of a tenant that requires company sign-in", context: member(EnforcementRequired, MFARequirementNone, connA), expectedOffered: true, expectedCompanyOnly: true},
		{name: "invited", context: invited(EnforcementOff), expectedAdmitted: true, expectedOffered: true},
		{name: "invited to a tenant that requires company sign-in", context: invited(EnforcementRequired, connA), expectedAdmitted: true, expectedOffered: true, expectedCompanyOnly: true},
		{name: "admitted by auto-join", context: autoJoinAdmitted(MFARequirementNone, connA), expectedAdmitted: true, expectedOffered: true, expectedCompanyOnly: true},
		{name: "member with an invitation", context: &SignInContext{Member: true, InvitationAdmits: true, Enforcement: EnforcementOff}, expectedOffered: true},
		{name: "neither member nor admitted", context: &SignInContext{Enforcement: EnforcementRequired}, expectedCompanyOnly: true},
		{name: "no context", context: noContext},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if admitted := tt.context.Admitted(); admitted != tt.expectedAdmitted {
				t.Fatalf("expected admitted %v, got %v", tt.expectedAdmitted, admitted)
			}
			if offered := tt.context.Offered(); offered != tt.expectedOffered {
				t.Fatalf("expected offered %v, got %v", tt.expectedOffered, offered)
			}
			if companyOnly := tt.context.CompanyOnly(); companyOnly != tt.expectedCompanyOnly {
				t.Fatalf("expected company only %v, got %v", tt.expectedCompanyOnly, companyOnly)
			}
		})
	}
}

func TestSignInContextApplies(t *testing.T) {
	var noContext *SignInContext

	tests := []struct {
		name         string
		context      *SignInContext
		connectionID string
		expected     bool
	}{
		{name: "active binding", context: member(EnforcementRequired, MFARequirementNone, connA), connectionID: connA, expected: true},
		{name: "another connection", context: member(EnforcementRequired, MFARequirementNone, connA), connectionID: connB, expected: false},
		{name: "no connection", context: member(EnforcementRequired, MFARequirementNone, connA), connectionID: "", expected: false},
		{name: "no context", context: noContext, connectionID: connA, expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if applies := tt.context.Applies(tt.connectionID); applies != tt.expected {
				t.Fatalf("expected %v, got %v", tt.expected, applies)
			}
		})
	}
}

func TestToSignInContext(t *testing.T) {
	tests := []struct {
		name        string
		context     *tenant.SignInContext
		expected    *SignInContext
		expectedErr bool
	}{
		{
			name: "member of a tenant that requires company sign-in and MFA",
			context: &tenant.SignInContext{
				Member:         true,
				Enforcement:    tenant.Enforcement_ENFORCEMENT_REQUIRED,
				ConnectionIds:  []string{connA},
				MfaRequirement: tenant.MFARequirement_MFA_REQUIREMENT_REQUIRED,
				AccountExists:  true,
			},
			expected: &SignInContext{Member: true, Enforcement: EnforcementRequired, ConnectionIDs: []string{connA}, MFARequirement: MFARequirementRequired, AccountExists: true},
		},
		{
			name: "address admitted by auto-join and an invitation",
			context: &tenant.SignInContext{
				Enforcement:      tenant.Enforcement_ENFORCEMENT_REQUIRED,
				ConnectionIds:    []string{connA},
				AutoJoinAdmits:   true,
				InvitationAdmits: true,
				MfaRequirement:   tenant.MFARequirement_MFA_REQUIREMENT_NONE,
			},
			expected: &SignInContext{Enforcement: EnforcementRequired, ConnectionIDs: []string{connA}, AutoJoinAdmits: true, InvitationAdmits: true, MFARequirement: MFARequirementNone},
		},
		{
			name:     "optional",
			context:  &tenant.SignInContext{Member: true, Enforcement: tenant.Enforcement_ENFORCEMENT_OPTIONAL, ConnectionIds: []string{connA}, MfaRequirement: tenant.MFARequirement_MFA_REQUIREMENT_NONE},
			expected: &SignInContext{Member: true, Enforcement: EnforcementOptional, ConnectionIDs: []string{connA}, MFARequirement: MFARequirementNone},
		},
		{
			// a tenant with no active binding offers no company sign-in
			name:     "off",
			context:  &tenant.SignInContext{Member: true, Enforcement: tenant.Enforcement_ENFORCEMENT_OFF, ConnectionIds: []string{connA}, MfaRequirement: tenant.MFARequirement_MFA_REQUIREMENT_NONE},
			expected: &SignInContext{Member: true, Enforcement: EnforcementOff, MFARequirement: MFARequirementNone},
		},
		// a value tenant-service did not set is not read as the weakest one
		{name: "unspecified enforcement", context: &tenant.SignInContext{Member: true, MfaRequirement: tenant.MFARequirement_MFA_REQUIREMENT_NONE}, expectedErr: true},
		{name: "unspecified MFA policy", context: &tenant.SignInContext{Member: true, Enforcement: tenant.Enforcement_ENFORCEMENT_OFF}, expectedErr: true},
		{name: "no context", context: nil, expectedErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc, err := toSignInContext(tt.context)

			if (err != nil) != tt.expectedErr {
				t.Fatalf("expected error %v, got %v", tt.expectedErr, err)
			}
			if !reflect.DeepEqual(sc, tt.expected) {
				t.Fatalf("expected sign-in context %+v, got %+v", tt.expected, sc)
			}
		})
	}
}

func TestDirectorySuccess(t *testing.T) {
	ctx := context.Background()
	// the order of the lookup is kept: the personal tenant comes first
	tenants := []SignInTenant{{ID: "personal", Name: "Bob"}, {ID: "acme", Name: "Acme", Invited: true}, {ID: "hooli", Name: "Hooli", AutoJoinCandidate: true}}

	// each call is made once, with a deadline, and its answer is returned
	tests := []struct {
		name     string
		span     string
		call     func(mockClient *MockTenantSignInServiceClientInterface, d *Directory) (any, error)
		expected any
	}{
		{name: "SignInTenants", span: "byosso.Directory.SignInTenants", expected: tenants, call: func(mockClient *MockTenantSignInServiceClientInterface, d *Directory) (any, error) {
			mockClient.EXPECT().ListSignInTenants(underBudget{}, &tenant.ListSignInTenantsRequest{Email: testEmail}).Times(1).Return(
				&tenant.ListSignInTenantsResponse{Tenants: []*tenant.SignInTenant{
					{Tenant: &tenant.Tenant{Id: "personal", Name: "Bob"}},
					{Tenant: &tenant.Tenant{Id: "acme", Name: "Acme"}, Invited: true},
					{Tenant: &tenant.Tenant{Id: "hooli", Name: "Hooli"}, AutoJoinCandidate: true},
				}}, nil,
			)
			return d.SignInTenants(ctx, testEmail)
		}},
		{name: "SignInContext", span: "byosso.Directory.SignInContext", expected: member(EnforcementOff, MFARequirementNone), call: func(mockClient *MockTenantSignInServiceClientInterface, d *Directory) (any, error) {
			mockClient.EXPECT().GetSignInContext(underBudget{}, &tenant.GetSignInContextRequest{TenantId: testTenant, IdentityId: "iid"}).Times(1).Return(
				&tenant.GetSignInContextResponse{Context: &tenant.SignInContext{Member: true, Enforcement: tenant.Enforcement_ENFORCEMENT_OFF, MfaRequirement: tenant.MFARequirement_MFA_REQUIREMENT_NONE}}, nil,
			)
			return d.SignInContext(ctx, testTenant, "", "iid")
		}},
		{name: "JoinTenant", span: "byosso.Directory.JoinTenant", call: func(mockClient *MockTenantSignInServiceClientInterface, d *Directory) (any, error) {
			mockClient.EXPECT().JoinTenant(underBudget{}, &tenant.JoinTenantRequest{TenantId: testTenant, IdentityId: "iid"}).Times(1).Return(&tenant.JoinTenantResponse{}, nil)
			return nil, d.JoinTenant(ctx, testTenant, "iid")
		}},
		{name: "CreatePersonalTenant", span: "byosso.Directory.CreatePersonalTenant", expected: "personal", call: func(mockClient *MockTenantSignInServiceClientInterface, d *Directory) (any, error) {
			mockClient.EXPECT().CreatePersonalTenant(underBudget{}, &tenant.CreatePersonalTenantRequest{IdentityId: "iid"}).Times(1).Return(
				&tenant.CreatePersonalTenantResponse{Tenant: &tenant.Tenant{Id: "personal"}}, nil,
			)
			return d.CreatePersonalTenant(ctx, "iid")
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockLogger := NewMockLoggerInterface(ctrl)
			mockClient := NewMockTenantSignInServiceClientInterface(ctrl)
			mockTracer := NewMockTracingInterface(ctrl)

			mockTracer.EXPECT().Start(ctx, tt.span).Times(1).Return(ctx, trace.SpanFromContext(ctx))

			got, err := tt.call(mockClient, NewDirectory(mockClient, time.Second, mockTracer, mockLogger))

			expectNoError(t, err)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Fatalf("expected %+v, got %+v", tt.expected, got)
			}
		})
	}
}

func TestDirectoryFails(t *testing.T) {
	ctx := context.Background()

	// expected is the refusal the error is, nil when it is none of them
	tests := []struct {
		name                string
		call                func(mockClient *MockTenantSignInServiceClientInterface, d *Directory) error
		expected            error
		expectedUnavailable bool
	}{
		{name: "not admitted", expected: errNotAdmitted, call: func(mockClient *MockTenantSignInServiceClientInterface, d *Directory) error {
			mockClient.EXPECT().JoinTenant(gomock.Any(), gomock.Any()).Times(1).Return(nil, statusWithReason(codes.FailedPrecondition, "NOT_ADMITTED"))
			return d.JoinTenant(ctx, testTenant, "iid")
		}},
		// an outage is not a refusal
		{name: "unavailable", expectedUnavailable: true, call: func(mockClient *MockTenantSignInServiceClientInterface, d *Directory) error {
			mockClient.EXPECT().JoinTenant(gomock.Any(), gomock.Any()).Times(1).Return(nil, status.Error(codes.Unavailable, "down"))
			return d.JoinTenant(ctx, testTenant, "iid")
		}},
		{name: "account with a tenant", expected: errHasTenant, call: func(mockClient *MockTenantSignInServiceClientInterface, d *Directory) error {
			mockClient.EXPECT().CreatePersonalTenant(gomock.Any(), gomock.Any()).Times(1).Return(nil, statusWithReason(codes.FailedPrecondition, "HAS_TENANT"))
			_, err := d.CreatePersonalTenant(ctx, "iid")
			return err
		}},
		{name: "unavailable when creating a personal tenant", expectedUnavailable: true, call: func(mockClient *MockTenantSignInServiceClientInterface, d *Directory) error {
			mockClient.EXPECT().CreatePersonalTenant(gomock.Any(), gomock.Any()).Times(1).Return(nil, status.Error(codes.Unavailable, "down"))
			_, err := d.CreatePersonalTenant(ctx, "iid")
			return err
		}},
		{name: "empty tenant", call: func(mockClient *MockTenantSignInServiceClientInterface, d *Directory) error {
			mockClient.EXPECT().CreatePersonalTenant(gomock.Any(), gomock.Any()).Times(1).Return(&tenant.CreatePersonalTenantResponse{}, nil)
			_, err := d.CreatePersonalTenant(ctx, "iid")
			return err
		}},
		// an enforcement login-ui does not know is not read as the weakest one
		{name: "unknown enforcement", call: func(mockClient *MockTenantSignInServiceClientInterface, d *Directory) error {
			mockClient.EXPECT().GetSignInContext(gomock.Any(), gomock.Any()).Times(1).Return(
				&tenant.GetSignInContextResponse{Context: &tenant.SignInContext{Member: true, Enforcement: 42}}, nil,
			)
			_, err := d.SignInContext(ctx, testTenant, "", "iid")
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockLogger := NewMockLoggerInterface(ctrl)
			mockClient := NewMockTenantSignInServiceClientInterface(ctrl)
			mockTracer := NewMockTracingInterface(ctrl)

			mockLogger.EXPECT().Debugf(gomock.Any(), gomock.Any()).AnyTimes()
			mockTracer.EXPECT().Start(ctx, gomock.Any()).Times(1).Return(ctx, trace.SpanFromContext(ctx))

			err := tt.call(mockClient, NewDirectory(mockClient, time.Second, mockTracer, mockLogger))

			if err == nil {
				t.Fatalf("expected error not nil")
			}
			for _, refusal := range []error{errNotAdmitted, errHasTenant} {
				if is := errors.Is(err, refusal); is != (refusal == tt.expected) {
					t.Fatalf("expected refusal %q to be %v, got %v", refusal, refusal == tt.expected, is)
				}
			}
			if unavailable := isUnavailable(err); unavailable != tt.expectedUnavailable {
				t.Fatalf("expected unavailable %v, got %v", tt.expectedUnavailable, unavailable)
			}
		})
	}
}
