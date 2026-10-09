// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package byosso

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	kClient "github.com/ory/kratos-client-go/v25"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/canonical/identity-platform-login-ui/internal/cookies"
	"github.com/canonical/identity-platform-login-ui/pkg/kratos"
)

// expectStateCookie expects the request to carry stateCookie, which binds
// tenantID to the login challenge.
func (m *apiMocks) expectStateCookie(stateCookie cookies.FlowStateCookie, tenantID string) {
	m.state.EXPECT().GetStateCookie(gomock.Any()).AnyTimes().Return(stateCookie, nil)
	m.tenants.EXPECT().TenantID(gomock.Any(), testLoginChallenge).AnyTimes().Return(tenantID)
}

// signInRequest returns the request of the login page carrying the sign-in
// cookie of a company sign-in.
func signInRequest(mocks *apiMocks, signIn SignInCookie) *http.Request {
	return withCookies(createLoginFlowRequest(), func(w http.ResponseWriter) {
		_ = mocks.store.SetSignIn(w, signIn)
	})
}

func TestCheckAndAcceptWithoutIdentity(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, _ := newTestAPI(t, ctrl)

	rec := httptest.NewRecorder()
	api.checkAndAccept(rec, createLoginFlowRequest(), kClient.NewSessionWithDefaults(), testLoginChallenge, stateFor(testTenant))

	expectStatus(t, rec, http.StatusUnauthorized)
}

func TestCheckAndAcceptCompletesCompanySignIn(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, mocks := newTestAPI(t, ctrl)
	mocks.store.now = func() time.Time { return t0 }
	session := companySession("s-new")
	expiresAt := t0.Add(8 * time.Hour)
	session.ExpiresAt = &expiresAt
	r := signInRequest(mocks, SignInCookie{Ticket: "ticket", PriorSessionID: "s-old", LoginChallengeHash: cookies.ChallengeHash(testLoginChallenge)})
	withReceipt(r, "ticket", "receipt")

	mocks.expectStateCookie(stateFor(testTenant), testTenant)
	mocks.sso.EXPECT().CompleteAttempt(gomock.Any(), "ticket", "iid", "receipt").Times(1).Return(&Completion{ConnectionID: connA, TenantID: testTenant}, nil)
	mocks.hydra.EXPECT().LoginRequest(underBudget{}, testLoginChallenge).Times(1).Return(&LoginRequest{}, nil)
	mocks.directory.EXPECT().SignInContext(gomock.Any(), testTenant, "", "iid").Times(1).Return(member(EnforcementRequired, MFARequirementNone, connA), nil)
	mocks.expectAccept(session, testTenant)

	rec := httptest.NewRecorder()
	api.HandleSessionLogin(rec, r, session, testLoginChallenge)

	expectStatus(t, rec, http.StatusOK)
	expectRedirectTo(t, rec, testAcceptRedirect)

	provenanceCookie := findCookie(rec, provenanceCookieName)
	provenance, err := mocks.store.GetProvenance(requestWith(provenanceCookie))
	expectNoError(t, err)
	if expected := (ProvenanceCookie{SessionID: "s-new", ConnectionID: connA}); provenance != expected {
		t.Fatalf("expected provenance cookie %+v, got %+v", expected, provenance)
	}
	if !provenanceCookie.Expires.Equal(expiresAt) {
		t.Fatalf("expected the provenance cookie to expire at %v, got %v", expiresAt, provenanceCookie.Expires)
	}
	if signInCookie := findCookie(rec, signInCookieName); signInCookie == nil || signInCookie.MaxAge >= 0 {
		t.Fatalf("expected the sign-in cookie to be cleared, got %v", signInCookie)
	}
	expectReceiptCleared(t, rec, "ticket")
}

func TestCheckAndAcceptCompletesAttemptBeingLinkedFirst(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, mocks := newTestAPI(t, ctrl)
	// the company sign-in the account had, then the one Kratos linked
	session := companySession("s-new", authenticationMethod("oidc", providerID, kClient.AUTHENTICATORASSURANCELEVEL_AAL1))
	r := signInRequest(mocks, SignInCookie{Ticket: "ticket-a", LinkTicket: "ticket-b", LoginChallengeHash: cookies.ChallengeHash(testLoginChallenge)})

	mocks.expectStateCookie(stateFor(testTenant), testTenant)
	mocks.sso.EXPECT().CompleteAttempt(gomock.Any(), "ticket-b", "iid", "").Times(1).Return(&Completion{ConnectionID: connB, TenantID: testTenant}, nil)
	mocks.hydra.EXPECT().LoginRequest(gomock.Any(), testLoginChallenge).Times(1).Return(&LoginRequest{}, nil)
	mocks.directory.EXPECT().SignInContext(gomock.Any(), testTenant, "", "iid").Times(1).Return(member(EnforcementRequired, MFARequirementNone, connB), nil)
	mocks.expectAccept(session, testTenant)

	rec := httptest.NewRecorder()
	api.HandleSessionLogin(rec, r, session, testLoginChallenge)

	expectStatus(t, rec, http.StatusOK)

	provenance, err := mocks.store.GetProvenance(requestWith(findCookie(rec, provenanceCookieName)))
	expectNoError(t, err)
	if provenance.ConnectionID != connB {
		t.Fatalf("expected provenance of connection %s, got %s", connB, provenance.ConnectionID)
	}
}

func TestCheckAndAcceptRestoresTenantOfTheAttempt(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, mocks := newTestAPI(t, ctrl)
	session := companySession("s-new")
	r := signInRequest(mocks, SignInCookie{Ticket: "ticket", LoginChallengeHash: cookies.ChallengeHash(testLoginChallenge)})

	// the state cookie expired while the user was at the IdP
	mocks.state.EXPECT().GetStateCookie(gomock.Any()).Times(1).Return(cookies.FlowStateCookie{}, nil)
	mocks.tenants.EXPECT().TenantID(cookies.FlowStateCookie{}, testLoginChallenge).Times(1).Return("")
	mocks.sso.EXPECT().CompleteAttempt(gomock.Any(), "ticket", "iid", "").Times(1).Return(&Completion{ConnectionID: connA, TenantID: testTenant}, nil)
	mocks.state.EXPECT().SetStateCookie(gomock.Any(), stateFor(testTenant)).Times(1).Return(nil)
	mocks.tenants.EXPECT().TenantID(stateFor(testTenant), testLoginChallenge).Times(1).Return(testTenant)
	mocks.hydra.EXPECT().LoginRequest(gomock.Any(), testLoginChallenge).Times(1).Return(&LoginRequest{}, nil)
	mocks.directory.EXPECT().SignInContext(gomock.Any(), testTenant, "", "iid").Times(1).Return(member(EnforcementRequired, MFARequirementNone, connA), nil)
	mocks.expectAccept(session, testTenant)

	rec := httptest.NewRecorder()
	api.HandleSessionLogin(rec, r, session, testLoginChallenge)

	expectStatus(t, rec, http.StatusOK)
	expectRedirectTo(t, rec, testAcceptRedirect)
}

func TestCheckAndAcceptWithSessionOfBeforeTheCompanySignIn(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, mocks := newTestAPI(t, ctrl)
	session := companySession("s-old")
	r := signInRequest(mocks, SignInCookie{Ticket: "ticket", PriorSessionID: "s-old"})

	// no attempt is completed for the session the browser already had
	mocks.expectStateCookie(stateFor(testTenant), testTenant)
	mocks.hydra.EXPECT().LoginRequest(gomock.Any(), testLoginChallenge).Times(1).Return(&LoginRequest{}, nil)
	mocks.directory.EXPECT().SignInContext(gomock.Any(), testTenant, "", "iid").Times(1).Return(member(EnforcementRequired, MFARequirementNone, connA), nil)
	mocks.sso.EXPECT().Options(gomock.Any(), []string{connA}).Times(1).Return([]Option{{ConnectionID: connA, Label: "Acme"}}, nil)
	mocks.expectNewLoginFlow(t)
	mocks.expectIdentify(t, nil)
	mocks.expectStartAttempt(t, "fresh", AttemptRequest{TenantID: testTenant, Email: testEmail, ConnectionID: connA})
	mocks.state.EXPECT().SetStateCookie(gomock.Any(), stateFor(testTenant)).Times(1).Return(nil)

	rec := httptest.NewRecorder()
	api.HandleSessionLogin(rec, r, session, testLoginChallenge)

	expectStatus(t, rec, http.StatusOK)
	expectRedirectTo(t, rec, testCompanySignIn)
}

func TestCheckAndAcceptAsksCompanySignIn(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, mocks := newTestAPI(t, ctrl)
	session := passwordSession("s1")

	mocks.expectStateCookie(stateFor(testTenant), testTenant)
	mocks.hydra.EXPECT().LoginRequest(gomock.Any(), testLoginChallenge).Times(1).Return(&LoginRequest{}, nil)
	mocks.directory.EXPECT().SignInContext(gomock.Any(), testTenant, "", "iid").Times(1).Return(member(EnforcementRequired, MFARequirementNone, connA), nil)
	mocks.sso.EXPECT().Options(gomock.Any(), []string{connA}).Times(1).Return([]Option{{ConnectionID: connA, Label: "Acme"}}, nil)
	mocks.expectNewLoginFlow(t)
	mocks.expectIdentify(t, nil)
	mocks.expectStartAttempt(t, "fresh", AttemptRequest{TenantID: testTenant, Email: testEmail, ConnectionID: connA})
	mocks.state.EXPECT().SetStateCookie(gomock.Any(), stateFor(testTenant)).Times(1).Return(nil)

	rec := httptest.NewRecorder()
	api.HandleSessionLogin(rec, createLoginFlowRequest(), session, testLoginChallenge)

	expectRedirectTo(t, rec, testCompanySignIn)
	if label := responseBody(t, rec)["redirect_label"]; label != "Acme" {
		t.Fatalf("expected redirect label Acme, got %v", label)
	}

	signIn, err := mocks.store.GetSignIn(requestWith(findCookie(rec, signInCookieName)))
	expectNoError(t, err)
	// the account does not join the tenant by this sign-in
	expected := SignInCookie{Ticket: "ticket-1", PriorSessionID: "s1", LoginChallengeHash: cookies.ChallengeHash(testLoginChallenge), TenantID: testTenant, ConnectionID: connA}
	if signIn != expected {
		t.Fatalf("expected sign-in cookie %+v, got %+v", expected, signIn)
	}
	// the IdP returns to Kratos, which must not find a session there
	if sessionCookie := findCookie(rec, kratos.KRATOS_SESSION_COOKIE_NAME); sessionCookie == nil || sessionCookie.Value != "" {
		t.Fatalf("expected the session cookie to be unset, got %v", sessionCookie)
	}
}

func TestCheckAndAcceptAsksCompanySignInWithoutOptions(t *testing.T) {
	tests := []struct {
		name            string
		err             error
		expectedStatus  int
		expectedErrorID string
	}{
		{name: "no company sign-in applies", err: nil, expectedStatus: http.StatusForbidden, expectedErrorID: ssoNotApplicableError},
		{name: "sso-service fails", err: errors.New("error"), expectedStatus: http.StatusServiceUnavailable, expectedErrorID: ssoUnavailableError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			api, mocks := newTestAPI(t, ctrl)

			mocks.expectStateCookie(stateFor(testTenant), testTenant)
			mocks.hydra.EXPECT().LoginRequest(gomock.Any(), testLoginChallenge).Times(1).Return(&LoginRequest{}, nil)
			mocks.directory.EXPECT().SignInContext(gomock.Any(), testTenant, "", "iid").Times(1).Return(member(EnforcementRequired, MFARequirementNone, connA), nil)
			mocks.sso.EXPECT().Options(gomock.Any(), []string{connA}).Times(1).Return(nil, tt.err)

			rec := httptest.NewRecorder()
			api.HandleSessionLogin(rec, createLoginFlowRequest(), passwordSession("s1"), testLoginChallenge)

			expectStatus(t, rec, tt.expectedStatus)
			expectErrorID(t, rec, tt.expectedErrorID)
		})
	}
}

func TestCheckAndAcceptRefusesNonMember(t *testing.T) {
	tests := []struct {
		name    string
		session *kClient.Session
		context *SignInContext
	}{
		{name: "password session", session: passwordSession("s1"), context: &SignInContext{Enforcement: EnforcementOff}},
		// auto-join was switched off while the user was at the IdP
		{name: "company session the tenant admits no more", session: companySession("s1"), context: &SignInContext{Enforcement: EnforcementRequired, ConnectionIDs: []string{connA}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			api, mocks := newTestAPI(t, ctrl)

			mocks.expectStateCookie(stateFor(testTenant), testTenant)
			mocks.hydra.EXPECT().LoginRequest(gomock.Any(), testLoginChallenge).Times(1).Return(&LoginRequest{}, nil)
			mocks.directory.EXPECT().SignInContext(gomock.Any(), testTenant, "", "iid").Times(1).Return(tt.context, nil)

			rec := httptest.NewRecorder()
			api.HandleSessionLogin(rec, createLoginFlowRequest(), tt.session, testLoginChallenge)

			expectStatus(t, rec, http.StatusForbidden)
			expectErrorID(t, rec, notAMemberError)
			expectErrorMessage(t, rec, notAMemberMessage)
			if sessionCookie := findCookie(rec, kratos.KRATOS_SESSION_COOKIE_NAME); sessionCookie != nil {
				t.Fatalf("expected the session cookie to be kept, got %v", sessionCookie)
			}
		})
	}
}

func TestCheckAndAcceptJoinsTenant(t *testing.T) {
	tests := []struct {
		name       string
		session    *kClient.Session
		provenance string
		context    *SignInContext
	}{
		{name: "invited with password", session: passwordSession("s1", totp()), context: invited(EnforcementOff)},
		{name: "invited with company sign-in", session: companySession("s1"), provenance: connA, context: invited(EnforcementRequired, connA)},
		{name: "admitted by auto-join with company sign-in", session: companySession("s1"), provenance: connA, context: autoJoinAdmitted(MFARequirementNone, connA)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			api, mocks := newTestAPI(t, ctrl)
			r := withCookies(createLoginFlowRequest(), func(w http.ResponseWriter) {
				if tt.provenance != "" {
					_ = mocks.store.SetProvenance(w, ProvenanceCookie{SessionID: "s1", ConnectionID: tt.provenance}, tt.session)
				}
			})

			mocks.expectStateCookie(stateFor(testTenant), testTenant)
			mocks.hydra.EXPECT().LoginRequest(gomock.Any(), testLoginChallenge).Times(1).Return(&LoginRequest{}, nil)
			mocks.directory.EXPECT().SignInContext(gomock.Any(), testTenant, "", "iid").Times(1).Return(tt.context, nil)
			join := mocks.directory.EXPECT().JoinTenant(gomock.Any(), testTenant, "iid").Times(1).Return(nil)
			redirectTo := testAcceptRedirect
			mocks.kratos.EXPECT().AcceptLoginRequest(gomock.Any(), tt.session, testLoginChallenge, testTenant).Times(1).After(join).Return(
				&kratos.BrowserLocationChangeRequired{RedirectTo: &redirectTo}, nil, nil,
			)
			mocks.state.EXPECT().ClearStateCookie(gomock.Any()).Times(1)

			rec := httptest.NewRecorder()
			api.HandleSessionLogin(rec, r, tt.session, testLoginChallenge)

			expectStatus(t, rec, http.StatusOK)
			expectRedirectTo(t, rec, testAcceptRedirect)
		})
	}
}

func TestCheckAndAcceptAsksCompanySignInOfAdmittingTenant(t *testing.T) {
	tests := []struct {
		name    string
		context *SignInContext
	}{
		{name: "invited", context: invited(EnforcementRequired, connA)},
		{name: "admitted by auto-join", context: autoJoinAdmitted(MFARequirementNone, connA)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			api, mocks := newTestAPI(t, ctrl)

			// the account does not join before it signs in with the company sign-in
			mocks.expectStateCookie(stateFor(testTenant), testTenant)
			mocks.hydra.EXPECT().LoginRequest(gomock.Any(), testLoginChallenge).Times(1).Return(&LoginRequest{}, nil)
			mocks.directory.EXPECT().SignInContext(gomock.Any(), testTenant, "", "iid").Times(1).Return(tt.context, nil)
			mocks.sso.EXPECT().Options(gomock.Any(), []string{connA}).Times(1).Return([]Option{{ConnectionID: connA, Label: "Acme"}}, nil)
			mocks.expectNewLoginFlow(t)
			mocks.expectIdentify(t, nil)
			mocks.expectStartAttempt(t, "fresh", AttemptRequest{TenantID: testTenant, Email: testEmail, ConnectionID: connA})
			mocks.state.EXPECT().SetStateCookie(gomock.Any(), stateFor(testTenant)).Times(1).Return(nil)

			rec := httptest.NewRecorder()
			api.HandleSessionLogin(rec, createLoginFlowRequest(), passwordSession("s1"), testLoginChallenge)

			expectRedirectTo(t, rec, testCompanySignIn)
		})
	}
}

func TestCheckAndAcceptFailOnJoinTenant(t *testing.T) {
	tests := []struct {
		name                 string
		err                  error
		expectedStatus       int
		expectedErrorID      string
		expectedErrorMessage string
	}{
		{
			// the tenant stopped admitting the address since the check
			name:                 "not admitted",
			err:                  fmt.Errorf("cannot join tenant: %w", errNotAdmitted),
			expectedStatus:       http.StatusForbidden,
			expectedErrorID:      notAMemberError,
			expectedErrorMessage: notAMemberMessage,
		},
		{
			name:                 "tenant-service unavailable",
			err:                  tenantServiceUnavailable(),
			expectedStatus:       http.StatusServiceUnavailable,
			expectedErrorID:      ssoUnavailableError,
			expectedErrorMessage: tenantsUnavailableMessage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			api, mocks := newTestAPI(t, ctrl)
			session := companySession("s1")
			r := withCookies(createLoginFlowRequest(), func(w http.ResponseWriter) {
				_ = mocks.store.SetProvenance(w, ProvenanceCookie{SessionID: "s1", ConnectionID: connA}, session)
			})

			mocks.expectStateCookie(stateFor(testTenant), testTenant)
			mocks.hydra.EXPECT().LoginRequest(gomock.Any(), testLoginChallenge).Times(1).Return(&LoginRequest{}, nil)
			mocks.directory.EXPECT().SignInContext(gomock.Any(), testTenant, "", "iid").Times(1).Return(autoJoinAdmitted(MFARequirementNone, connA), nil)
			mocks.directory.EXPECT().JoinTenant(gomock.Any(), testTenant, "iid").Times(1).Return(tt.err)

			rec := httptest.NewRecorder()
			api.HandleSessionLogin(rec, r, session, testLoginChallenge)

			expectStatus(t, rec, tt.expectedStatus)
			expectErrorID(t, rec, tt.expectedErrorID)
			expectErrorMessage(t, rec, tt.expectedErrorMessage)
			// the account the company sign-in registered stays, and so does its session
			if sessionCookie := findCookie(rec, kratos.KRATOS_SESSION_COOKIE_NAME); sessionCookie != nil {
				t.Fatalf("expected the session cookie to be kept, got %v", sessionCookie)
			}
		})
	}
}

func TestCheckAndAcceptWithCompanySessionAndNoTenant(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, mocks := newTestAPI(t, ctrl)
	redirectTo := "/ui/login?flow=fresh"

	// a company sign-in is never accepted at a personal tenant
	mocks.expectStateCookie(stateFor(cookies.NoTenantAvailable), cookies.NoTenantAvailable)
	mocks.hydra.EXPECT().LoginRequest(gomock.Any(), testLoginChallenge).Times(1).Return(&LoginRequest{}, nil)
	mocks.expectNewLoginFlow(t)
	mocks.expectIdentify(t, &kratos.BrowserLocationChangeRequired{RedirectTo: &redirectTo})
	mocks.state.EXPECT().SetStateCookie(gomock.Any(), stateFor(cookies.NoTenantAvailable)).Times(1).Return(nil)

	rec := httptest.NewRecorder()
	api.HandleSessionLogin(rec, createLoginFlowRequest(), companySession("s1"), testLoginChallenge)

	expectRedirectTo(t, rec, redirectTo)
}

func TestCheckAndAcceptWithPendingInvitationsAndNoTenant(t *testing.T) {
	tests := []struct {
		name             string
		tenants          []SignInTenant
		expectedState    cookies.FlowStateCookie
		expectedRedirect string
		expectedErrorID  string
	}{
		{
			// the invitation is accepted by this sign-in
			name:             "one pending invitation",
			tenants:          []SignInTenant{{ID: testTenant, Invited: true}},
			expectedState:    stateFor(testTenant),
			expectedRedirect: testAcceptRedirect,
		},
		{
			name:             "several pending invitations",
			tenants:          []SignInTenant{{ID: "a", Invited: true}, {ID: "b", Invited: true}},
			expectedState:    cookies.FlowStateCookie{TenantChoice: true},
			expectedRedirect: "/ui/select_tenant?login_challenge=" + testLoginChallenge,
			expectedErrorID:  tenantSelectionRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			api, mocks := newTestAPI(t, ctrl)
			session := passwordSession("s1", totp())

			// the address has pending invitations: no personal tenant is created
			mocks.expectStateCookie(stateFor(cookies.NoTenantAvailable), cookies.NoTenantAvailable)
			mocks.hydra.EXPECT().LoginRequest(gomock.Any(), testLoginChallenge).Times(1).Return(&LoginRequest{}, nil)
			mocks.directory.EXPECT().CreatePersonalTenant(gomock.Any(), "iid").Times(1).Return("", errHasTenant)
			mocks.directory.EXPECT().SignInTenants(gomock.Any(), testEmail).Times(1).Return(tt.tenants, nil)
			mocks.state.EXPECT().SetStateCookie(gomock.Any(), tt.expectedState).Times(1).Return(nil)
			if len(tt.tenants) == 1 {
				mocks.directory.EXPECT().SignInContext(gomock.Any(), testTenant, "", "iid").Times(1).Return(invited(EnforcementOff), nil)
				mocks.directory.EXPECT().JoinTenant(gomock.Any(), testTenant, "iid").Times(1).Return(nil)
				mocks.expectAccept(session, testTenant)
			}

			rec := httptest.NewRecorder()
			api.HandleSessionLogin(rec, createLoginFlowRequest(), session, testLoginChallenge)

			expectStatus(t, rec, http.StatusOK)
			expectRedirectTo(t, rec, tt.expectedRedirect)
			expectErrorID(t, rec, tt.expectedErrorID)
		})
	}
}

func TestCheckAndAcceptWithoutTenant(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("several tenants", func(t *testing.T) {
		api, mocks := newTestAPI(t, ctrl)

		mocks.expectStateCookie(cookies.FlowStateCookie{}, "")
		mocks.hydra.EXPECT().LoginRequest(gomock.Any(), testLoginChallenge).Times(1).Return(&LoginRequest{}, nil)
		mocks.directory.EXPECT().SignInTenants(gomock.Any(), testEmail).Times(1).Return([]SignInTenant{{ID: "a"}, {ID: "b"}}, nil)
		// the tenant picked on the tenant page is one chosen from several
		mocks.state.EXPECT().SetStateCookie(gomock.Any(), cookies.FlowStateCookie{TenantChoice: true}).Times(1).Return(nil)

		rec := httptest.NewRecorder()
		api.HandleSessionLogin(rec, createLoginFlowRequest(), passwordSession("s1"), testLoginChallenge)

		expectErrorID(t, rec, tenantSelectionRequired)
		expectedRedirect := "/ui/select_tenant?login_challenge=" + testLoginChallenge
		expectRedirectTo(t, rec, expectedRedirect)
	})

	t.Run("one tenant", func(t *testing.T) {
		api, mocks := newTestAPI(t, ctrl)
		session := passwordSession("s1", totp())

		mocks.state.EXPECT().GetStateCookie(gomock.Any()).Times(1).Return(cookies.FlowStateCookie{}, nil)
		mocks.tenants.EXPECT().TenantID(cookies.FlowStateCookie{}, testLoginChallenge).Times(1).Return("")
		mocks.hydra.EXPECT().LoginRequest(gomock.Any(), testLoginChallenge).Times(1).Return(&LoginRequest{}, nil)
		mocks.directory.EXPECT().SignInTenants(gomock.Any(), testEmail).Times(1).Return([]SignInTenant{{ID: testTenant}}, nil)
		mocks.state.EXPECT().SetStateCookie(gomock.Any(), stateFor(testTenant)).Times(1).Return(nil)
		mocks.directory.EXPECT().SignInContext(gomock.Any(), testTenant, "", "iid").Times(1).Return(member(EnforcementOff, MFARequirementNone), nil)
		mocks.expectAccept(session, testTenant)

		rec := httptest.NewRecorder()
		api.HandleSessionLogin(rec, createLoginFlowRequest(), session, testLoginChallenge)

		expectRedirectTo(t, rec, testAcceptRedirect)
	})

	t.Run("no tenant", func(t *testing.T) {
		api, mocks := newTestAPI(t, ctrl)
		session := passwordSession("s1", totp())

		mocks.state.EXPECT().GetStateCookie(gomock.Any()).Times(1).Return(cookies.FlowStateCookie{}, nil)
		mocks.tenants.EXPECT().TenantID(cookies.FlowStateCookie{}, testLoginChallenge).Times(1).Return("")
		mocks.hydra.EXPECT().LoginRequest(gomock.Any(), testLoginChallenge).Times(1).Return(&LoginRequest{}, nil)
		mocks.directory.EXPECT().SignInTenants(gomock.Any(), testEmail).Times(1).Return(nil, nil)
		mocks.state.EXPECT().SetStateCookie(gomock.Any(), stateFor(cookies.NoTenantAvailable)).Times(1).Return(nil)
		mocks.directory.EXPECT().CreatePersonalTenant(gomock.Any(), "iid").Times(1).Return("personal-1", nil)
		mocks.directory.EXPECT().SignInContext(gomock.Any(), "personal-1", "", "iid").Times(1).Return(member(EnforcementOff, MFARequirementNone), nil)
		redirectTo := testAcceptRedirect
		mocks.kratos.EXPECT().AcceptLoginRequest(gomock.Any(), session, testLoginChallenge, "personal-1").Times(1).Return(
			&kratos.BrowserLocationChangeRequired{RedirectTo: &redirectTo}, nil, nil,
		)
		mocks.state.EXPECT().ClearStateCookie(gomock.Any()).Times(1)

		rec := httptest.NewRecorder()
		api.HandleSessionLogin(rec, createLoginFlowRequest(), session, testLoginChallenge)

		expectRedirectTo(t, rec, testAcceptRedirect)
	})
}

func TestCheckAndAcceptRevokesLoginSessionOfAnotherSubject(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, mocks := newTestAPI(t, ctrl)
	requestURL := "https://login.example/oauth2/auth?client_id=app&state=s"

	// nothing is accepted: the request of the app starts again
	mocks.expectStateCookie(stateFor(testTenant), testTenant)
	mocks.hydra.EXPECT().LoginRequest(gomock.Any(), testLoginChallenge).Times(1).Return(&LoginRequest{Subject: "someone-else", SessionID: "sid-1", RequestURL: requestURL}, nil)
	mocks.hydra.EXPECT().RevokeLoginSession(gomock.Any(), "sid-1").Times(1).Return(nil)

	rec := httptest.NewRecorder()
	api.HandleSessionLogin(rec, createLoginFlowRequest(), passwordSession("s1"), testLoginChallenge)

	expectStatus(t, rec, http.StatusOK)
	expectRedirectTo(t, rec, requestURL)
}

func TestCheckAndAcceptAsksFreshFirstFactor(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, mocks := newTestAPI(t, ctrl)
	redirectTo := "/ui/login?flow=fresh"

	// the app asks for a fresh sign-in, and the session is from before its request
	mocks.expectStateCookie(stateFor(testTenant), testTenant)
	mocks.hydra.EXPECT().LoginRequest(gomock.Any(), testLoginChallenge).Times(1).Return(&LoginRequest{PromptLogin: true}, nil)
	mocks.directory.EXPECT().SignInContext(gomock.Any(), testTenant, "", "iid").Times(1).Return(member(EnforcementOff, MFARequirementNone), nil)
	mocks.expectNewLoginFlow(t)
	mocks.expectIdentify(t, &kratos.BrowserLocationChangeRequired{RedirectTo: &redirectTo})
	mocks.state.EXPECT().SetStateCookie(gomock.Any(), stateFor(testTenant)).Times(1).Return(nil)

	rec := httptest.NewRecorder()
	api.HandleSessionLogin(rec, createLoginFlowRequest(), passwordSession("s1"), testLoginChallenge)

	expectRedirectTo(t, rec, redirectTo)
}

func TestCheckAndAcceptWithFreshFirstFactor(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, mocks := newTestAPI(t, ctrl)
	session := passwordSession("s2", totp())
	r := withCookies(createLoginFlowRequest(), func(w http.ResponseWriter) {
		_ = mocks.store.SetFresh(w, FreshCookie{LoginChallengeHash: cookies.ChallengeHash(testLoginChallenge), PriorSessionID: "s1", StartedAt: t0})
	})

	mocks.expectStateCookie(stateFor(testTenant), testTenant)
	mocks.hydra.EXPECT().LoginRequest(gomock.Any(), testLoginChallenge).Times(1).Return(&LoginRequest{PromptLogin: true}, nil)
	mocks.directory.EXPECT().SignInContext(gomock.Any(), testTenant, "", "iid").Times(1).Return(member(EnforcementOff, MFARequirementNone), nil)
	mocks.expectAccept(session, testTenant)

	rec := httptest.NewRecorder()
	api.HandleSessionLogin(rec, r, session, testLoginChallenge)

	expectRedirectTo(t, rec, testAcceptRedirect)
	if freshCookie := findCookie(rec, freshCookieName); freshCookie == nil || freshCookie.MaxAge >= 0 {
		t.Fatalf("expected the fresh cookie to be cleared, got %v", freshCookie)
	}
}

func TestCheckAndAcceptFailOnSignInContext(t *testing.T) {
	tests := []struct {
		name                 string
		err                  error
		expectedErrorMessage string
	}{
		{name: "unavailable", err: status.Error(codes.Unavailable, "down"), expectedErrorMessage: tenantsUnavailableMessage},
		{name: "deadline exceeded", err: status.Error(codes.DeadlineExceeded, "slow"), expectedErrorMessage: tenantsUnavailableMessage},
		{name: "not found", err: status.Error(codes.NotFound, "no such tenant"), expectedErrorMessage: ssoUnavailableMessage},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			api, mocks := newTestAPI(t, ctrl)

			mocks.expectStateCookie(stateFor(testTenant), testTenant)
			mocks.hydra.EXPECT().LoginRequest(gomock.Any(), testLoginChallenge).Times(1).Return(&LoginRequest{}, nil)
			mocks.directory.EXPECT().SignInContext(gomock.Any(), testTenant, "", "iid").Times(1).Return(nil, tt.err)

			rec := httptest.NewRecorder()
			api.HandleSessionLogin(rec, createLoginFlowRequest(), passwordSession("s1"), testLoginChallenge)

			expectStatus(t, rec, http.StatusServiceUnavailable)
			expectErrorID(t, rec, ssoUnavailableError)
			expectErrorMessage(t, rec, tt.expectedErrorMessage)
		})
	}
}

// unverifiedCompanySession builds the session of an account Kratos
// registered through a company sign-in that did not confirm the address.
func unverifiedCompanySession() *kClient.Session {
	session := companySession("s-new")
	session.Identity.VerifiableAddresses = []kClient.VerifiableIdentityAddress{
		*kClient.NewVerifiableIdentityAddress("pending", testEmail, false, "email"),
	}
	return session
}

func TestCheckAndAcceptVerifiesAddress(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, mocks := newTestAPI(t, ctrl)
	returnTo := BASE_URL + "/ui/login?login_challenge=" + testLoginChallenge

	// the address is verified before the attempt is completed and anything is accepted
	mocks.state.EXPECT().GetStateCookie(gomock.Any()).Times(1).Return(stateFor(testTenant), nil)
	mocks.verification.EXPECT().Start(underBudget{}, returnTo, testEmail, gomock.Any()).Times(1).Return("vf-1", []*http.Cookie{{Name: "csrf_token", Value: "csrf"}}, nil)

	rec := httptest.NewRecorder()
	api.HandleSessionLogin(rec, createLoginFlowRequest(), unverifiedCompanySession(), testLoginChallenge)

	expectStatus(t, rec, http.StatusOK)
	expectRedirectTo(t, rec, "/ui/verification?flow=vf-1")
	expectErrorID(t, rec, kratos.VERIFICATION_REQUIRED)
	if c := findCookie(rec, "csrf_token"); c == nil {
		t.Fatalf("expected the cookies of Kratos to be set")
	}
}

func TestCheckAndAcceptFails(t *testing.T) {
	tests := []struct {
		name                 string
		session              *kClient.Session
		signIn               *SignInCookie
		setupMocks           func(mocks *apiMocks, session *kClient.Session)
		expectedStatus       int
		expectedErrorID      string
		expectedErrorMessage string
	}{
		{
			name:    "verification start",
			session: unverifiedCompanySession(),
			setupMocks: func(mocks *apiMocks, _ *kClient.Session) {
				mocks.state.EXPECT().GetStateCookie(gomock.Any()).Times(1).Return(stateFor(testTenant), nil)
				mocks.verification.EXPECT().Start(gomock.Any(), gomock.Any(), testEmail, gomock.Any()).Times(1).Return("", nil, errors.New("error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:    "complete attempt",
			session: companySession("s-new"),
			signIn:  &SignInCookie{Ticket: "ticket"},
			setupMocks: func(mocks *apiMocks, _ *kClient.Session) {
				mocks.expectStateCookie(stateFor(testTenant), testTenant)
				mocks.sso.EXPECT().CompleteAttempt(gomock.Any(), "ticket", "iid", "").Times(1).Return(nil, errors.New("error"))
			},
			expectedStatus:       http.StatusServiceUnavailable,
			expectedErrorID:      ssoUnavailableError,
			expectedErrorMessage: ssoUnavailableMessage,
		},
		{
			name:    "login request",
			session: passwordSession("s1"),
			setupMocks: func(mocks *apiMocks, _ *kClient.Session) {
				mocks.expectStateCookie(stateFor(testTenant), testTenant)
				mocks.hydra.EXPECT().LoginRequest(gomock.Any(), testLoginChallenge).Times(1).Return(nil, errors.New("error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:    "revoke login session",
			session: passwordSession("s1"),
			setupMocks: func(mocks *apiMocks, _ *kClient.Session) {
				mocks.expectStateCookie(stateFor(testTenant), testTenant)
				mocks.hydra.EXPECT().LoginRequest(gomock.Any(), testLoginChallenge).Times(1).Return(&LoginRequest{Subject: "someone-else", SessionID: "sid-1", RequestURL: "https://login.example/oauth2/auth"}, nil)
				mocks.hydra.EXPECT().RevokeLoginSession(gomock.Any(), "sid-1").Times(1).Return(errors.New("error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:    "lookup by email",
			session: passwordSession("s1"),
			setupMocks: func(mocks *apiMocks, _ *kClient.Session) {
				mocks.expectStateCookie(cookies.FlowStateCookie{}, "")
				mocks.hydra.EXPECT().LoginRequest(gomock.Any(), testLoginChallenge).Times(1).Return(&LoginRequest{}, nil)
				mocks.directory.EXPECT().SignInTenants(gomock.Any(), testEmail).Times(1).Return(nil, tenantServiceUnavailable())
			},
			expectedStatus:       http.StatusServiceUnavailable,
			expectedErrorID:      ssoUnavailableError,
			expectedErrorMessage: tenantsUnavailableMessage,
		},
		{
			name:    "create personal tenant",
			session: passwordSession("s1"),
			setupMocks: func(mocks *apiMocks, _ *kClient.Session) {
				mocks.expectStateCookie(stateFor(cookies.NoTenantAvailable), cookies.NoTenantAvailable)
				mocks.hydra.EXPECT().LoginRequest(gomock.Any(), testLoginChallenge).Times(1).Return(&LoginRequest{}, nil)
				mocks.directory.EXPECT().CreatePersonalTenant(gomock.Any(), "iid").Times(1).Return("", tenantServiceUnavailable())
			},
			expectedStatus:       http.StatusServiceUnavailable,
			expectedErrorID:      ssoUnavailableError,
			expectedErrorMessage: tenantsUnavailableMessage,
		},
		{
			name:    "accept login request",
			session: passwordSession("s1", totp()),
			setupMocks: func(mocks *apiMocks, session *kClient.Session) {
				mocks.expectStateCookie(stateFor(testTenant), testTenant)
				mocks.hydra.EXPECT().LoginRequest(gomock.Any(), testLoginChallenge).Times(1).Return(&LoginRequest{}, nil)
				mocks.directory.EXPECT().SignInContext(gomock.Any(), testTenant, "", "iid").Times(1).Return(member(EnforcementOff, MFARequirementNone), nil)
				mocks.kratos.EXPECT().AcceptLoginRequest(gomock.Any(), session, testLoginChallenge, testTenant).Times(1).Return(nil, nil, errors.New("error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			api, mocks := newTestAPI(t, ctrl)
			r := createLoginFlowRequest()
			if tt.signIn != nil {
				r = signInRequest(mocks, *tt.signIn)
			}

			tt.setupMocks(mocks, tt.session)

			rec := httptest.NewRecorder()
			api.HandleSessionLogin(rec, r, tt.session, testLoginChallenge)

			expectStatus(t, rec, tt.expectedStatus)
			if tt.expectedErrorID != "" {
				expectErrorID(t, rec, tt.expectedErrorID)
				expectErrorMessage(t, rec, tt.expectedErrorMessage)
			}
			// the attempt can be completed again
			if c := findCookie(rec, signInCookieName); tt.signIn != nil && c != nil {
				t.Fatalf("expected the sign-in cookie to be kept, got %v", c)
			}
		})
	}
}
