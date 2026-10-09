// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package byosso

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	kClient "github.com/ory/kratos-client-go/v25"
	"go.uber.org/mock/gomock"

	"github.com/canonical/identity-platform-login-ui/internal/cookies"
	"github.com/canonical/identity-platform-login-ui/pkg/kratos"
)

func boolPointer(b bool) *bool {
	return &b
}

// expectChosenTenant expects stateCookie to bind testTenant to the login
// challenge, and the tenant to answer context for testEmail.
func (m *apiMocks) expectChosenTenant(stateCookie cookies.FlowStateCookie, context *SignInContext) {
	m.state.EXPECT().GetStateCookie(gomock.Any()).AnyTimes().Return(stateCookie, nil)
	m.tenants.EXPECT().TenantID(gomock.Any(), testLoginChallenge).AnyTimes().Return(testTenant)
	m.directory.EXPECT().SignInContext(underBudget{}, testTenant, testEmail, "").Times(1).Return(context, nil)
}

// expectNoTenantChosen expects stateCookie to bind no tenant to the login
// challenge.
func (m *apiMocks) expectNoTenantChosen(stateCookie cookies.FlowStateCookie) {
	m.state.EXPECT().GetStateCookie(gomock.Any()).AnyTimes().Return(stateCookie, nil)
	m.tenants.EXPECT().TenantID(gomock.Any(), testLoginChallenge).AnyTimes().Return("")
}

func TestHydrateChosenTenant(t *testing.T) {
	acme := Option{ConnectionID: connA, Label: "Acme"}
	legacy := Option{ConnectionID: connB, Label: "Acme legacy"}

	tests := []struct {
		name             string
		stateCookie      cookies.FlowStateCookie
		context          *SignInContext
		options          []Option
		expectedPassword int
		expectedPublic   int
		expectedCompany  int
		expectedReset    bool
	}{
		{
			name:            "required",
			stateCookie:     stateFor(testTenant),
			context:         member(EnforcementRequired, MFARequirementNone, connA, connB),
			options:         []Option{acme, legacy},
			expectedCompany: 2,
		},
		{
			name:             "optional",
			stateCookie:      stateFor(testTenant),
			context:          member(EnforcementOptional, MFARequirementNone, connA),
			options:          []Option{acme},
			expectedPassword: 1,
			expectedPublic:   1,
			expectedCompany:  1,
		},
		{
			name:             "off",
			stateCookie:      stateFor(testTenant),
			context:          member(EnforcementOff, MFARequirementNone),
			expectedPassword: 1,
			expectedPublic:   1,
		},
		{
			name:            "admitted by auto-join",
			stateCookie:     stateFor(testTenant),
			context:         autoJoinAdmitted(MFARequirementNone, connA),
			options:         []Option{acme},
			expectedCompany: 1,
		},
		{
			name:            "invited at required",
			stateCookie:     stateFor(testTenant),
			context:         invited(EnforcementRequired, connA),
			options:         []Option{acme},
			expectedCompany: 1,
		},
		{
			name:             "invited at optional",
			stateCookie:      stateFor(testTenant),
			context:          invited(EnforcementOptional, connA),
			options:          []Option{acme},
			expectedPassword: 1,
			expectedPublic:   1,
			expectedCompany:  1,
		},
		{
			name:             "chosen from several memberships",
			stateCookie:      chosenStateFor(testTenant),
			context:          member(EnforcementOptional, MFARequirementNone, connA),
			options:          []Option{acme},
			expectedPassword: 1,
			expectedPublic:   1,
			expectedCompany:  1,
			expectedReset:    true,
		},
		{
			name:            "chosen from several invitations",
			stateCookie:     chosenStateFor(testTenant),
			context:         invited(EnforcementRequired, connA),
			options:         []Option{acme},
			expectedCompany: 1,
			expectedReset:   true,
		},
		{
			name:            "chosen from several auto-join candidates",
			stateCookie:     chosenStateFor(testTenant),
			context:         autoJoinAdmitted(MFARequirementNone, connA),
			options:         []Option{acme},
			expectedCompany: 1,
			expectedReset:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			api, mocks := newTestAPI(t, ctrl)

			mocks.expectChosenTenant(tt.stateCookie, tt.context)
			mocks.sso.EXPECT().Options(gomock.Any(), tt.context.ConnectionIDs).Times(1).Return(tt.options, nil)

			flow, ok := api.HydrateLoginFlow(httptest.NewRecorder(), getLoginFlowRequest(), loginFlow("f", testLoginChallenge, testEmail, false))

			expectRendered(t, ok)
			expectNodes(t, flow, "password", tt.expectedPassword)
			expectNodes(t, flow, "oidc", tt.expectedPublic)
			company := nodesOf(flow, ssoNodeGroup)
			if len(company) != tt.expectedCompany {
				t.Fatalf("expected %d company sign-in nodes, got %d", tt.expectedCompany, len(company))
			}
			for i, node := range company {
				attrs := node.Attributes.UiNodeInputAttributes
				if attrs.Name != "sso_connection" {
					t.Fatalf("expected input sso_connection, got %s", attrs.Name)
				}
				if attrs.Value != tt.options[i].ConnectionID {
					t.Fatalf("expected value %s, got %v", tt.options[i].ConnectionID, attrs.Value)
				}
				if label := node.Meta.Label.Text; label != "Continue with "+tt.options[i].Label {
					t.Fatalf("expected label Continue with %s, got %s", tt.options[i].Label, label)
				}
			}
			if reset := hasNode(flow, "sso_tenant_reset"); reset != tt.expectedReset {
				t.Fatalf("expected the way back to the tenant list to be %v, got %v", tt.expectedReset, reset)
			}
		})
	}
}

func TestHydrateChosenTenantOfRefusedAddress(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, mocks := newTestAPI(t, ctrl)

	// an address with no account, which no identity is looked up for
	mocks.expectChosenTenant(stateFor(testTenant), invited(EnforcementRequired, connA))
	mocks.sso.EXPECT().Options(gomock.Any(), []string{connA}).Times(1).Return([]Option{{ConnectionID: connA, Label: "Acme"}}, nil)

	flow, ok := api.HydrateLoginFlow(httptest.NewRecorder(), getLoginFlowRequest(), refusedLoginFlow())

	expectRendered(t, ok)
	expectNodes(t, flow, ssoNodeGroup, 1)
	if hasIdentifierFirst(flow) {
		t.Fatalf("expected the identifier step to be removed")
	}
}

func TestHydrateChosenTenantWithoutOptions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, mocks := newTestAPI(t, ctrl)

	// the address is outside the domains of the tenant
	mocks.expectChosenTenant(stateFor(testTenant), member(EnforcementRequired, MFARequirementNone))
	mocks.sso.EXPECT().Options(gomock.Any(), gomock.Nil()).Times(1).Return(nil, nil)

	rec := httptest.NewRecorder()
	_, ok := api.HydrateLoginFlow(rec, getLoginFlowRequest(), loginFlow("f", testLoginChallenge, testEmail, false))

	if ok {
		t.Fatalf("expected the request to be answered")
	}
	expectStatus(t, rec, http.StatusForbidden)
	expectErrorID(t, rec, ssoNotApplicableError)
}

func TestHydrateChosenTenantFailOnOptions(t *testing.T) {
	t.Run("optional", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		api, mocks := newTestAPI(t, ctrl)

		mocks.expectChosenTenant(stateFor(testTenant), member(EnforcementOptional, MFARequirementNone, connA))
		mocks.sso.EXPECT().Options(gomock.Any(), []string{connA}).Times(1).Return(nil, errors.New("error"))

		flow, ok := api.HydrateLoginFlow(httptest.NewRecorder(), getLoginFlowRequest(), loginFlow("f", testLoginChallenge, testEmail, false))

		expectRendered(t, ok)
		expectNodes(t, flow, "password", 1)
		if !hasMessage(flow, companyUnavailableID) {
			t.Fatalf("expected the message that company sign-in is unavailable")
		}
	})

	t.Run("required", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		api, mocks := newTestAPI(t, ctrl)

		mocks.expectChosenTenant(stateFor(testTenant), member(EnforcementRequired, MFARequirementNone, connA))
		mocks.sso.EXPECT().Options(gomock.Any(), []string{connA}).Times(1).Return(nil, errors.New("error"))

		rec := httptest.NewRecorder()
		_, ok := api.HydrateLoginFlow(rec, getLoginFlowRequest(), loginFlow("f", testLoginChallenge, testEmail, false))

		if ok {
			t.Fatalf("expected the request to be answered")
		}
		expectStatus(t, rec, http.StatusServiceUnavailable)
		expectErrorID(t, rec, ssoUnavailableError)
	})
}

func TestHydrateChosenTenantFailOnSignInContext(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, mocks := newTestAPI(t, ctrl)

	mocks.state.EXPECT().GetStateCookie(gomock.Any()).Times(1).Return(stateFor(testTenant), nil)
	mocks.tenants.EXPECT().TenantID(gomock.Any(), testLoginChallenge).Times(1).Return(testTenant)
	mocks.directory.EXPECT().SignInContext(gomock.Any(), testTenant, testEmail, "").Times(1).Return(nil, tenantServiceUnavailable())

	rec := httptest.NewRecorder()
	_, ok := api.HydrateLoginFlow(rec, getLoginFlowRequest(), loginFlow("f", testLoginChallenge, testEmail, false))

	if ok {
		t.Fatalf("expected the request to be answered")
	}
	expectStatus(t, rec, http.StatusServiceUnavailable)
	expectErrorMessage(t, rec, tenantsUnavailableMessage)
}

func TestHydrateChosenTenantNotOffered(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, mocks := newTestAPI(t, ctrl)

	// the tenant is bound for another address: the tenants of this one are listed
	mocks.expectChosenTenant(stateFor(testTenant), &SignInContext{Enforcement: EnforcementOff})
	mocks.directory.EXPECT().SignInTenants(gomock.Any(), testEmail).Times(1).Return([]SignInTenant{{ID: "a", Name: "Alpha"}, {ID: "b", Name: "Beta"}}, nil)

	flow, ok := api.HydrateLoginFlow(httptest.NewRecorder(), getLoginFlowRequest(), loginFlow("f", testLoginChallenge, testEmail, false))

	expectRendered(t, ok)
	expectNodes(t, flow, tenantNodeGroup, 2)
}

func TestHydrateSignInScreen(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, mocks := newTestAPI(t, ctrl)

	mocks.expectNoTenantChosen(cookies.FlowStateCookie{TenantChoice: true})
	mocks.directory.EXPECT().SignInTenants(underBudget{}, testEmail).Times(1).Return([]SignInTenant{{ID: "personal", Name: "Bob"}, {ID: testTenant, Name: "Acme", Invited: true}}, nil)

	flow, ok := api.HydrateLoginFlow(httptest.NewRecorder(), getLoginFlowRequest(), loginFlow("f", testLoginChallenge, testEmail, false))

	expectRendered(t, ok)
	// the sign-ins of the account are shown once a tenant is picked
	expectNodes(t, flow, "password", 0)
	expectNodes(t, flow, "oidc", 0)

	tenants := nodesOf(flow, tenantNodeGroup)
	if len(tenants) != 2 {
		t.Fatalf("expected 2 tenant nodes, got %d", len(tenants))
	}
	expected := []struct{ id, label string }{{id: "personal", label: "Bob"}, {id: testTenant, label: "Acme — invitation"}}
	for i, node := range tenants {
		attrs := node.Attributes.UiNodeInputAttributes
		if attrs.Name != "sso_tenant" {
			t.Fatalf("expected input sso_tenant, got %s", attrs.Name)
		}
		if attrs.Value != expected[i].id {
			t.Fatalf("expected value %s, got %v", expected[i].id, attrs.Value)
		}
		if label := node.Meta.Label.Text; label != expected[i].label {
			t.Fatalf("expected label %s, got %s", expected[i].label, label)
		}
	}
}

func TestHydrateSignInScreenWithoutTenants(t *testing.T) {
	tests := []struct {
		name                  string
		flow                  func() *kClient.LoginFlow
		expectedPassword      int
		expectedRecoverPrompt bool
	}{
		{
			name:             "account with a password",
			flow:             func() *kClient.LoginFlow { return loginFlow("f", testLoginChallenge, testEmail, false) },
			expectedPassword: 1,
		},
		{
			name: "account with nothing to sign in with",
			flow: func() *kClient.LoginFlow {
				flow := loginFlow("f", testLoginChallenge, testEmail, false)
				flow.Ui.Nodes = flow.Ui.Nodes[:2]
				return flow
			},
			expectedRecoverPrompt: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			api, mocks := newTestAPI(t, ctrl)

			mocks.expectNoTenantChosen(cookies.FlowStateCookie{})
			mocks.directory.EXPECT().SignInTenants(gomock.Any(), testEmail).Times(1).Return(nil, nil)

			flow, ok := api.HydrateLoginFlow(httptest.NewRecorder(), getLoginFlowRequest(), tt.flow())

			expectRendered(t, ok)
			expectNodes(t, flow, "password", tt.expectedPassword)
			if prompt := hasMessage(flow, recoverPromptID); prompt != tt.expectedRecoverPrompt {
				t.Fatalf("expected the recovery prompt to be %v, got %v", tt.expectedRecoverPrompt, prompt)
			}
		})
	}
}

func TestHydrateSignInScreenFailOnSignInTenants(t *testing.T) {
	tests := []struct {
		name            string
		flow            func() *kClient.LoginFlow
		expectedMessage bool
	}{
		{
			// the account can still sign in with what it has
			name: "account with a password",
			flow: func() *kClient.LoginFlow { return loginFlow("f", testLoginChallenge, testEmail, false) },
		},
		{
			name: "account with nothing to sign in with",
			flow: func() *kClient.LoginFlow {
				flow := loginFlow("f", testLoginChallenge, testEmail, false)
				flow.Ui.Nodes = flow.Ui.Nodes[:2]
				return flow
			},
			expectedMessage: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			api, mocks := newTestAPI(t, ctrl)

			mocks.expectNoTenantChosen(cookies.FlowStateCookie{})
			mocks.directory.EXPECT().SignInTenants(gomock.Any(), testEmail).Times(1).Return(nil, errors.New("error"))

			flow, ok := api.HydrateLoginFlow(httptest.NewRecorder(), getLoginFlowRequest(), tt.flow())

			expectRendered(t, ok)
			if message := hasMessage(flow, lookupFailedID); message != tt.expectedMessage {
				t.Fatalf("expected the message that the tenants could not be loaded to be %v, got %v", tt.expectedMessage, message)
			}
		})
	}
}

func TestHydrateSignInScreenAtIdentifierStep(t *testing.T) {
	tests := []struct {
		name  string
		email string
	}{
		{name: "with an address", email: testEmail},
		{name: "without address", email: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			api, mocks := newTestAPI(t, ctrl)

			// no tenant is looked up before the address is submitted
			mocks.expectNoTenantChosen(cookies.FlowStateCookie{})

			flow, ok := api.HydrateLoginFlow(httptest.NewRecorder(), getLoginFlowRequest(), loginFlow("f", testLoginChallenge, tt.email, true))

			expectRendered(t, ok)
			if !hasIdentifierFirst(flow) {
				t.Fatalf("expected the identifier step to be kept")
			}
			expectNodes(t, flow, tenantNodeGroup, 0)
		})
	}
}

func TestHydrateSignInScreenOfRefusedAddress(t *testing.T) {
	tests := []struct {
		name                   string
		tenants                []SignInTenant
		lookupErr              error
		identityExists         *bool
		expectedTenants        int
		expectedIdentifierStep bool
		expectedMessage        int64
	}{
		{
			// whoever the address is: no identity is looked up
			name:            "tenants",
			tenants:         []SignInTenant{{ID: "a", Invited: true}, {ID: "b", AutoJoinCandidate: true}},
			expectedTenants: 2,
		},
		{
			name:            "no tenant and an account",
			identityExists:  boolPointer(true),
			expectedTenants: 1,
			expectedMessage: recoverPromptID,
		},
		{
			name:                   "no tenant and no account",
			identityExists:         boolPointer(false),
			expectedIdentifierStep: true,
			expectedMessage:        kratos.IncorrectAccountIdentifier,
		},
		{
			name:            "lookup fails and an account",
			lookupErr:       errors.New("error"),
			identityExists:  boolPointer(true),
			expectedMessage: lookupFailedID,
		},
		{
			name:                   "lookup fails and no account",
			lookupErr:              errors.New("error"),
			identityExists:         boolPointer(false),
			expectedIdentifierStep: true,
			expectedMessage:        kratos.IncorrectAccountIdentifier,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			api, mocks := newTestAPI(t, ctrl)

			mocks.expectNoTenantChosen(cookies.FlowStateCookie{})
			mocks.directory.EXPECT().SignInTenants(gomock.Any(), testEmail).Times(1).Return(tt.tenants, tt.lookupErr)
			if tt.identityExists != nil {
				mocks.identities.EXPECT().IdentityExists(gomock.Any(), testEmail).Times(1).Return(*tt.identityExists, nil)
			}

			flow, ok := api.HydrateLoginFlow(httptest.NewRecorder(), getLoginFlowRequest(), refusedLoginFlow())

			expectRendered(t, ok)
			expectNodes(t, flow, tenantNodeGroup, tt.expectedTenants)
			if identifierStep := hasIdentifierFirst(flow); identifierStep != tt.expectedIdentifierStep {
				t.Fatalf("expected the identifier step to be %v, got %v", tt.expectedIdentifierStep, identifierStep)
			}
			if tt.expectedMessage != 0 && !hasMessage(flow, tt.expectedMessage) {
				t.Fatalf("expected message %d, got %+v", tt.expectedMessage, flow.Ui.Messages)
			}
		})
	}
}

func TestHydrateNoTenant(t *testing.T) {
	tests := []struct {
		name                   string
		flow                   func() *kClient.LoginFlow
		identityExists         *bool
		expectedPassword       int
		expectedIdentifierStep bool
		expectedRecoverPrompt  bool
	}{
		{
			name:             "account with a password",
			flow:             func() *kClient.LoginFlow { return loginFlow("f", testLoginChallenge, testEmail, false) },
			expectedPassword: 1,
		},
		{
			name: "account with nothing to sign in with",
			flow: func() *kClient.LoginFlow {
				flow := loginFlow("f", testLoginChallenge, testEmail, false)
				flow.Ui.Nodes = flow.Ui.Nodes[:2]
				return flow
			},
			expectedRecoverPrompt: true,
		},
		{
			name:                  "address Kratos refused and an account",
			flow:                  refusedLoginFlow,
			identityExists:        boolPointer(true),
			expectedRecoverPrompt: true,
		},
		{
			name:                   "address Kratos refused and no account",
			flow:                   refusedLoginFlow,
			identityExists:         boolPointer(false),
			expectedIdentifierStep: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			api, mocks := newTestAPI(t, ctrl)

			mocks.state.EXPECT().GetStateCookie(gomock.Any()).Times(1).Return(stateFor(cookies.NoTenantAvailable), nil)
			mocks.tenants.EXPECT().TenantID(gomock.Any(), testLoginChallenge).Times(1).Return(cookies.NoTenantAvailable)
			if tt.identityExists != nil {
				mocks.identities.EXPECT().IdentityExists(gomock.Any(), testEmail).Times(1).Return(*tt.identityExists, nil)
			}

			flow, ok := api.HydrateLoginFlow(httptest.NewRecorder(), getLoginFlowRequest(), tt.flow())

			expectRendered(t, ok)
			expectNodes(t, flow, "password", tt.expectedPassword)
			if identifierStep := hasIdentifierFirst(flow); identifierStep != tt.expectedIdentifierStep {
				t.Fatalf("expected the identifier step to be %v, got %v", tt.expectedIdentifierStep, identifierStep)
			}
			if prompt := hasMessage(flow, recoverPromptID); prompt != tt.expectedRecoverPrompt {
				t.Fatalf("expected the recovery prompt to be %v, got %v", tt.expectedRecoverPrompt, prompt)
			}
		})
	}
}

func pickRequest(connectionID string) *http.Request {
	return updateLoginFlowRequest(fmt.Sprintf(`{"sso_connection":%q,"csrf_token":"csrf-1"}`, connectionID))
}

func TestPick(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, mocks := newTestAPI(t, ctrl)
	flow := loginFlow("f", testLoginChallenge, testEmail, false)
	requestURL := "https://hydra.example/oauth2/auth?prompt=login"
	flow.Oauth2LoginRequest.RequestUrl = &requestURL

	mocks.expectChosenTenant(stateFor(testTenant), autoJoinAdmitted(MFARequirementNone, connA))
	mocks.expectStartAttempt(t, "f", AttemptRequest{TenantID: testTenant, Email: testEmail, ConnectionID: connA, Reauthenticate: true})
	mocks.state.EXPECT().SetStateCookie(gomock.Any(), stateFor(testTenant)).Times(1).Return(nil)

	rec := httptest.NewRecorder()
	_, answered := api.InterceptLoginSubmission(rec, pickRequest(connA), flow)

	expectAnswered(t, answered)
	expectRedirectTo(t, rec, testCompanySignIn)

	fresh, err := mocks.store.GetFresh(requestWith(findCookie(rec, freshCookieName)))
	expectNoError(t, err)
	if fresh.LoginChallengeHash != cookies.ChallengeHash(testLoginChallenge) {
		t.Fatalf("expected the fresh cookie of the login challenge, got %+v", fresh)
	}
}

func TestPickWithSession(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, mocks := newTestAPI(t, ctrl)
	flow := loginFlow("f", testLoginChallenge, testEmail, false)
	requestURL := "https://hydra.example/oauth2/auth?max_age=3600"
	flow.Oauth2LoginRequest.RequestUrl = &requestURL

	// the first factor of the session is one minute old: the IdP is not asked for a fresh login
	mocks.expectChosenTenant(stateFor(testTenant), member(EnforcementOptional, MFARequirementNone, connA))
	mocks.kratos.EXPECT().CheckSession(gomock.Any(), gomock.Any()).Times(1).Return(passwordSession("s1"), nil, nil)
	mocks.expectStartAttempt(t, "f", AttemptRequest{TenantID: testTenant, Email: testEmail, ConnectionID: connA})
	mocks.state.EXPECT().SetStateCookie(gomock.Any(), stateFor(testTenant)).Times(1).Return(nil)

	rec := httptest.NewRecorder()
	_, answered := api.InterceptLoginSubmission(rec, withSessionCookie(pickRequest(connA)), flow)

	expectAnswered(t, answered)

	signIn, err := mocks.store.GetSignIn(requestWith(findCookie(rec, signInCookieName)))
	expectNoError(t, err)
	if signIn.PriorSessionID != "s1" {
		t.Fatalf("expected prior session s1, got %s", signIn.PriorSessionID)
	}
}

func TestPickRefused(t *testing.T) {
	tests := []struct {
		name           string
		connectionID   string
		setupMocks     func(mocks *apiMocks)
		expectedStatus int
	}{
		{
			name:         "connection not offered",
			connectionID: connB,
			setupMocks: func(mocks *apiMocks) {
				mocks.expectChosenTenant(stateFor(testTenant), member(EnforcementOptional, MFARequirementNone, connA))
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:         "no tenant",
			connectionID: connA,
			setupMocks: func(mocks *apiMocks) {
				mocks.state.EXPECT().GetStateCookie(gomock.Any()).Times(1).Return(cookies.FlowStateCookie{}, nil)
				mocks.tenants.EXPECT().TenantID(gomock.Any(), testLoginChallenge).Times(1).Return(cookies.NoTenantAvailable)
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			api, mocks := newTestAPI(t, ctrl)

			tt.setupMocks(mocks)

			rec := httptest.NewRecorder()
			_, answered := api.InterceptLoginSubmission(rec, pickRequest(tt.connectionID), loginFlow("f", testLoginChallenge, testEmail, false))

			expectAnswered(t, answered)
			expectStatus(t, rec, tt.expectedStatus)
		})
	}
}

func TestPickWithoutStateCookie(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, mocks := newTestAPI(t, ctrl)

	// the state cookie has expired: the login page lists the tenants again
	mocks.expectNoTenantChosen(cookies.FlowStateCookie{})

	rec := httptest.NewRecorder()
	_, answered := api.InterceptLoginSubmission(rec, pickRequest(connA), loginFlow("f", testLoginChallenge, testEmail, false))

	expectAnswered(t, answered)
	expectRedirectTo(t, rec, "/ui/login?flow=f")
}

func chooseTenantRequest(tenantID string) *http.Request {
	return updateLoginFlowRequest(fmt.Sprintf(`{"sso_tenant":%q,"csrf_token":"csrf-1"}`, tenantID))
}

func TestChooseTenant(t *testing.T) {
	tests := []struct {
		name          string
		stateCookie   cookies.FlowStateCookie
		expectedState cookies.FlowStateCookie
	}{
		{name: "from several tenants", stateCookie: cookies.FlowStateCookie{TenantChoice: true}, expectedState: chosenStateFor(testTenant)},
		{name: "without a choice of tenants", stateCookie: cookies.FlowStateCookie{}, expectedState: stateFor(testTenant)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			api, mocks := newTestAPI(t, ctrl)

			mocks.expectNoTenantChosen(tt.stateCookie)
			mocks.directory.EXPECT().SignInContext(gomock.Any(), testTenant, testEmail, "").Times(1).Return(member(EnforcementOptional, MFARequirementNone, connA), nil)
			mocks.state.EXPECT().SetStateCookie(gomock.Any(), tt.expectedState).Times(1).Return(nil)

			rec := httptest.NewRecorder()
			_, answered := api.InterceptLoginSubmission(rec, chooseTenantRequest(testTenant), loginFlow("f", testLoginChallenge, testEmail, false))

			expectAnswered(t, answered)
			// the login page shows what the tenant offers
			expectRedirectTo(t, rec, "/ui/login?flow=f")
		})
	}
}

func TestChooseTenantRequiringCompanySignIn(t *testing.T) {
	tests := []struct {
		name    string
		context *SignInContext
	}{
		{name: "member", context: member(EnforcementRequired, MFARequirementNone, connA)},
		{name: "invited", context: invited(EnforcementRequired, connA)},
		{name: "admitted by auto-join", context: autoJoinAdmitted(MFARequirementNone, connA)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			api, mocks := newTestAPI(t, ctrl)

			// the browser goes straight to the only company sign-in, and the tenant stays the chosen one
			mocks.expectNoTenantChosen(cookies.FlowStateCookie{TenantChoice: true})
			mocks.directory.EXPECT().SignInContext(gomock.Any(), testTenant, testEmail, "").Times(1).Return(tt.context, nil)
			mocks.sso.EXPECT().Options(gomock.Any(), []string{connA}).Times(1).Return([]Option{{ConnectionID: connA, Label: "Acme"}}, nil)
			mocks.expectStartAttempt(t, "f", AttemptRequest{TenantID: testTenant, Email: testEmail, ConnectionID: connA})
			mocks.state.EXPECT().SetStateCookie(gomock.Any(), chosenStateFor(testTenant)).Times(1).Return(nil)

			rec := httptest.NewRecorder()
			_, answered := api.InterceptLoginSubmission(rec, chooseTenantRequest(testTenant), loginFlow("f", testLoginChallenge, testEmail, false))

			expectAnswered(t, answered)
			expectRedirectTo(t, rec, testCompanySignIn)
			if label := responseBody(t, rec)["redirect_label"]; label != "Acme" {
				t.Fatalf("expected redirect label Acme, got %v", label)
			}
		})
	}
}

func TestChooseTenantRequiringCompanySignInWithSeveralOptions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, mocks := newTestAPI(t, ctrl)

	mocks.expectNoTenantChosen(cookies.FlowStateCookie{TenantChoice: true})
	mocks.directory.EXPECT().SignInContext(gomock.Any(), testTenant, testEmail, "").Times(1).Return(member(EnforcementRequired, MFARequirementNone, connA, connB), nil)
	mocks.sso.EXPECT().Options(gomock.Any(), []string{connA, connB}).Times(1).Return([]Option{{ConnectionID: connA, Label: "Acme"}, {ConnectionID: connB, Label: "Acme legacy"}}, nil)
	mocks.state.EXPECT().SetStateCookie(gomock.Any(), chosenStateFor(testTenant)).Times(1).Return(nil)

	rec := httptest.NewRecorder()
	_, answered := api.InterceptLoginSubmission(rec, chooseTenantRequest(testTenant), loginFlow("f", testLoginChallenge, testEmail, false))

	expectAnswered(t, answered)
	expectRedirectTo(t, rec, "/ui/login?flow=f")
}

func TestChooseTenantRequiringCompanySignInWithoutOptions(t *testing.T) {
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

			mocks.expectNoTenantChosen(cookies.FlowStateCookie{TenantChoice: true})
			mocks.directory.EXPECT().SignInContext(gomock.Any(), testTenant, testEmail, "").Times(1).Return(member(EnforcementRequired, MFARequirementNone, connA), nil)
			mocks.sso.EXPECT().Options(gomock.Any(), []string{connA}).Times(1).Return(nil, tt.err)

			rec := httptest.NewRecorder()
			_, answered := api.InterceptLoginSubmission(rec, chooseTenantRequest(testTenant), loginFlow("f", testLoginChallenge, testEmail, false))

			expectAnswered(t, answered)
			expectStatus(t, rec, tt.expectedStatus)
			expectErrorID(t, rec, tt.expectedErrorID)
		})
	}
}

func TestChooseTenantNotOffered(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, mocks := newTestAPI(t, ctrl)

	mocks.expectNoTenantChosen(cookies.FlowStateCookie{TenantChoice: true})
	mocks.directory.EXPECT().SignInContext(gomock.Any(), testTenant, testEmail, "").Times(1).Return(&SignInContext{Enforcement: EnforcementOff}, nil)

	rec := httptest.NewRecorder()
	_, answered := api.InterceptLoginSubmission(rec, chooseTenantRequest(testTenant), loginFlow("f", testLoginChallenge, testEmail, false))

	expectAnswered(t, answered)
	expectStatus(t, rec, http.StatusForbidden)
}

func TestChooseAndResetTenantWithoutLoginChallenge(t *testing.T) {
	tests := []struct {
		name    string
		request *http.Request
		flow    *kClient.LoginFlow
	}{
		{name: "tenant on a flow without login challenge", request: chooseTenantRequest(testTenant), flow: loginFlow("f", "", testEmail, false)},
		{name: "no tenant", request: chooseTenantRequest(cookies.NoTenantAvailable), flow: loginFlow("f", testLoginChallenge, testEmail, false)},
		{name: "reset on a flow without login challenge", request: updateLoginFlowRequest(`{"sso_tenant_reset":"1"}`), flow: loginFlow("f", "", testEmail, false)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			api, _ := newTestAPI(t, ctrl)

			rec := httptest.NewRecorder()
			_, answered := api.InterceptLoginSubmission(rec, tt.request, tt.flow)

			expectAnswered(t, answered)
			expectStatus(t, rec, http.StatusBadRequest)
		})
	}
}

func TestResetTenant(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, mocks := newTestAPI(t, ctrl)

	// nothing is chosen any more, and the user still has several tenants to choose from
	mocks.state.EXPECT().GetStateCookie(gomock.Any()).Times(1).Return(chosenStateFor(testTenant), nil)
	mocks.state.EXPECT().SetStateCookie(gomock.Any(), cookies.FlowStateCookie{TenantChoice: true}).Times(1).Return(nil)

	rec := httptest.NewRecorder()
	_, answered := api.InterceptLoginSubmission(rec, updateLoginFlowRequest(`{"sso_tenant_reset":"1","csrf_token":"csrf-1"}`), loginFlow("f", testLoginChallenge, testEmail, false))

	expectAnswered(t, answered)
	expectRedirectTo(t, rec, "/ui/login?flow=f")
}

func TestBindTenant(t *testing.T) {
	tests := []struct {
		name          string
		stateCookie   cookies.FlowStateCookie
		bound         string
		chosen        bool
		expectedState cookies.FlowStateCookie
	}{
		{name: "tenant picked from several", stateCookie: cookies.FlowStateCookie{TenantChoice: true}, chosen: true, expectedState: chosenStateFor(testTenant)},
		{name: "tenant bound as picked from several", stateCookie: chosenStateFor(testTenant), bound: testTenant, expectedState: chosenStateFor(testTenant)},
		{name: "another tenant bound as picked from several", stateCookie: chosenStateFor("another"), bound: "another", expectedState: stateFor(testTenant)},
		{name: "choice of tenants of another sign-in", stateCookie: cookies.FlowStateCookie{TenantChoice: true}, expectedState: stateFor(testTenant)},
		{name: "tenant bound", stateCookie: stateFor(testTenant), bound: testTenant, expectedState: stateFor(testTenant)},
		{name: "no state", stateCookie: cookies.FlowStateCookie{}, expectedState: stateFor(testTenant)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			api, mocks := newTestAPI(t, ctrl)

			mocks.state.EXPECT().GetStateCookie(gomock.Any()).Times(1).Return(tt.stateCookie, nil)
			mocks.tenants.EXPECT().TenantID(tt.stateCookie, testLoginChallenge).AnyTimes().Return(tt.bound)
			mocks.state.EXPECT().SetStateCookie(gomock.Any(), tt.expectedState).Times(1).Return(nil)

			err := api.bindTenant(httptest.NewRecorder(), getLoginFlowRequest(), testLoginChallenge, testTenant, tt.chosen)

			expectNoError(t, err)
		})
	}
}
