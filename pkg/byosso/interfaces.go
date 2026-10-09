// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package byosso

import (
	"context"
	"net/http"

	kClient "github.com/ory/kratos-client-go/v25"
	"google.golang.org/grpc"

	sso "github.com/canonical/identity-platform-api/v0/sso"
	tenant "github.com/canonical/identity-platform-api/v0/tenant"

	"github.com/canonical/identity-platform-login-ui/internal/cookies"
	"github.com/canonical/identity-platform-login-ui/internal/hydra"
	"github.com/canonical/identity-platform-login-ui/pkg/kratos"
)

type HydraClientInterface interface {
	OAuth2API() hydra.OAuth2API
}

type KratosClientInterface interface {
	FrontendApi() kClient.FrontendAPI
}

type KratosAdminClientInterface interface {
	IdentityApi() kClient.IdentityAPI
}

// SSOServiceClientInterface is the subset of the generated gRPC
// SSOSignInServiceClient needed by SSO.
type SSOServiceClientInterface interface {
	ListOptions(ctx context.Context, in *sso.ListOptionsRequest, opts ...grpc.CallOption) (*sso.ListOptionsResponse, error)
	StartAttempt(ctx context.Context, in *sso.StartAttemptRequest, opts ...grpc.CallOption) (*sso.StartAttemptResponse, error)
	CompleteAttempt(ctx context.Context, in *sso.CompleteAttemptRequest, opts ...grpc.CallOption) (*sso.CompleteAttemptResponse, error)
	ListLinks(ctx context.Context, in *sso.ListLinksRequest, opts ...grpc.CallOption) (*sso.ListLinksResponse, error)
	DeleteLink(ctx context.Context, in *sso.DeleteLinkRequest, opts ...grpc.CallOption) (*sso.DeleteLinkResponse, error)
}

// TenantSignInServiceClientInterface is the generated gRPC
// TenantSignInServiceClient, which Directory calls.
type TenantSignInServiceClientInterface interface {
	ListSignInTenants(ctx context.Context, in *tenant.ListSignInTenantsRequest, opts ...grpc.CallOption) (*tenant.ListSignInTenantsResponse, error)
	GetSignInContext(ctx context.Context, in *tenant.GetSignInContextRequest, opts ...grpc.CallOption) (*tenant.GetSignInContextResponse, error)
	JoinTenant(ctx context.Context, in *tenant.JoinTenantRequest, opts ...grpc.CallOption) (*tenant.JoinTenantResponse, error)
	CreatePersonalTenant(ctx context.Context, in *tenant.CreatePersonalTenantRequest, opts ...grpc.CallOption) (*tenant.CreatePersonalTenantResponse, error)
}

type SSOInterface interface {
	// Options returns the tested connections among connectionIDs.
	Options(ctx context.Context, connectionIDs []string) ([]Option, error)
	// StartAttempt returns the ticket of a new sign-in attempt.
	StartAttempt(ctx context.Context, req AttemptRequest) (string, error)
	// CompleteAttempt confirms that the attempt of the ticket ended at the
	// account. receipt is the receipt cookie of the ticket as the browser
	// sent it, "" when it sent none.
	CompleteAttempt(ctx context.Context, ticket, identityID, receipt string) (*Completion, error)
	Links(ctx context.Context, identityID string) ([]Link, error)
	DeleteLink(ctx context.Context, connectionID, identityID string) error
}

type DirectoryInterface interface {
	// SignInTenants returns the tenants of the address, the personal tenant
	// first, then the ones a pending invitation or auto-join admits it to.
	SignInTenants(ctx context.Context, email string) ([]SignInTenant, error)
	// SignInContext is asked by email before sign-in, by identityID with a
	// session.
	SignInContext(ctx context.Context, tenantID, email, identityID string) (*SignInContext, error)
	CreatePersonalTenant(ctx context.Context, identityID string) (string, error)
	JoinTenant(ctx context.Context, tenantID, identityID string) error
}

type HydraInterface interface {
	LoginRequest(ctx context.Context, loginChallenge string) (*LoginRequest, error)
	RejectConsent(ctx context.Context, consentChallenge, description string) (string, error)
	RevokeLoginSession(ctx context.Context, sid string) error
}

type VerificationInterface interface {
	// Start creates a verification flow that returns to returnTo and sends
	// the code to email. It returns the ID of the flow and the cookies of
	// Kratos.
	Start(ctx context.Context, returnTo, email string, cookies []*http.Cookie) (string, []*http.Cookie, error)
}

type IdentityFinderInterface interface {
	IdentityExists(ctx context.Context, email string) (bool, error)
	IdentityID(ctx context.Context, email string) (string, error)
	HasRecoveryCodes(ctx context.Context, identityID string) (bool, error)
}

// KratosServiceInterface is the subset of kratos.ServiceInterface needed by
// the API.
type KratosServiceInterface interface {
	CheckSession(context.Context, []*http.Cookie) (*kClient.Session, []*http.Cookie, error)
	AcceptLoginRequest(context.Context, *kClient.Session, string, string) (*kratos.BrowserLocationChangeRequired, []*http.Cookie, error)
	CreateBrowserLoginFlow(context.Context, string, string, string, bool, []*http.Cookie) (*kClient.LoginFlow, []*http.Cookie, error)
	GetRegistrationFlow(context.Context, string, []*http.Cookie) (*kClient.RegistrationFlow, []*http.Cookie, error)
	GetSettingsFlow(context.Context, string, []*http.Cookie) (*kClient.SettingsFlow, *kratos.BrowserLocationChangeRequired, error)
	UpdateLoginFlow(context.Context, string, kClient.UpdateLoginFlowBody, []*http.Cookie) (*kratos.BrowserLocationChangeRequired, *kClient.SuccessfulNativeLogin, []*http.Cookie, error)
	UpdateIdentifierFirstLoginFlow(context.Context, string, kClient.UpdateLoginFlowWithIdentifierFirstMethod, []*http.Cookie) (*kratos.BrowserLocationChangeRequired, []*http.Cookie, error)
	HasTOTPAvailable(context.Context, string) (bool, error)
	HasWebAuthnAvailable(context.Context, string) (bool, error)
}

// StateCookieInterface is the subset of the cookie manager needed by the API.
type StateCookieInterface interface {
	SetStateCookie(http.ResponseWriter, cookies.FlowStateCookie) error
	GetStateCookie(*http.Request) (cookies.FlowStateCookie, error)
	ClearStateCookie(http.ResponseWriter)
}

// TenantResolverInterface is the subset of kratos.TenantResolverInterface
// needed by the API.
type TenantResolverInterface interface {
	TenantID(cookie cookies.FlowStateCookie, loginChallenge string) string
}

type CookieStoreInterface interface {
	SetSignIn(http.ResponseWriter, SignInCookie) error
	GetSignIn(*http.Request) (SignInCookie, error)
	ClearSignIn(http.ResponseWriter)
	SetProvenance(http.ResponseWriter, ProvenanceCookie, *kClient.Session) error
	GetProvenance(*http.Request) (ProvenanceCookie, error)
	SetFresh(http.ResponseWriter, FreshCookie) error
	GetFresh(*http.Request) (FreshCookie, error)
	ClearFresh(http.ResponseWriter)
	SetRegistration(http.ResponseWriter, RegistrationCookie) error
	GetRegistration(*http.Request) (RegistrationCookie, error)
	ClearRegistration(http.ResponseWriter)
}
