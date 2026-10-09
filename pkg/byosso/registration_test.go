// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package byosso

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	kClient "github.com/ory/kratos-client-go/v25"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/canonical/identity-platform-login-ui/internal/cookies"
)

const testRegistrationEmail = "hank@hooli.example"

func registrationRequest() *http.Request {
	return jsonRequest(http.MethodPost, "/api/kratos/self-service/registration?flow=reg", `{"method":"password","traits":{"email":"hank@hooli.example"}}`)
}

// expectRegistrationLoginFlow expects the login flow of a registration with
// no login challenge to be created, returning to completePath.
func (m *apiMocks) expectRegistrationLoginFlow(registrationReturnTo string) {
	registrationFlow := kClient.NewRegistrationFlowWithDefaults()
	returnTo := BASE_URL + completePath
	if registrationReturnTo != "" {
		registrationFlow.SetReturnTo(registrationReturnTo)
		returnTo = BASE_URL + completePath + "?return_to=https%3A%2F%2Fportal.example%2Fwelcome"
	}
	m.kratos.EXPECT().GetRegistrationFlow(gomock.Any(), "reg", gomock.Any()).Times(1).Return(registrationFlow, nil, nil)
	m.kratos.EXPECT().CreateBrowserLoginFlow(gomock.Any(), "", returnTo, "", false, gomock.Any()).Times(1).Return(loginFlow("lf", "", "", true), nil, nil)
}

func TestRegistrationSignIn(t *testing.T) {
	tests := []struct {
		name    string
		tenant  SignInTenant
		context *SignInContext
	}{
		{name: "admitted by auto-join", tenant: SignInTenant{ID: testTenant, AutoJoinCandidate: true}, context: autoJoinAdmitted(MFARequirementNone, connA)},
		{name: "invited with no account", tenant: SignInTenant{ID: testTenant, Invited: true}, context: invited(EnforcementRequired, connA)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			api, mocks := newTestAPI(t, ctrl)

			mocks.directory.EXPECT().SignInTenants(gomock.Any(), testRegistrationEmail).Times(1).Return([]SignInTenant{tt.tenant}, nil)
			mocks.directory.EXPECT().SignInContext(gomock.Any(), testTenant, testRegistrationEmail, "").Times(1).Return(tt.context, nil)
			mocks.sso.EXPECT().Options(gomock.Any(), []string{connA}).Times(1).Return([]Option{{ConnectionID: connA, Label: "Hooli"}}, nil)
			mocks.expectRegistrationLoginFlow("https://portal.example/welcome")
			mocks.expectStartAttempt(t, "lf", AttemptRequest{TenantID: testTenant, Email: testRegistrationEmail, ConnectionID: connA})

			rec := httptest.NewRecorder()
			answered := api.InterceptRegistrationSubmission(rec, registrationRequest(), "reg")

			expectAnswered(t, answered)
			expectRedirectTo(t, rec, testCompanySignIn)
			if label := responseBody(t, rec)["redirect_label"]; label != "Hooli" {
				t.Fatalf("expected redirect label Hooli, got %v", label)
			}
			if responseBody(t, rec)["continue_with"] == nil {
				t.Fatalf("expected continue_with to be set")
			}

			signIn, err := mocks.store.GetSignIn(requestWith(findCookie(rec, signInCookieName)))
			expectNoError(t, err)
			// no login challenge, and the account joins the tenant
			expected := SignInCookie{Ticket: "ticket-1", ReturnTo: "https://portal.example/welcome", TenantID: testTenant, ConnectionID: connA, Join: true}
			if signIn != expected {
				t.Fatalf("expected sign-in cookie %+v, got %+v", expected, signIn)
			}
		})
	}
}

func TestRegistrationSignInWithLoginChallenge(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, mocks := newTestAPI(t, ctrl)
	registrationFlow := kClient.NewRegistrationFlowWithDefaults()
	registrationFlow.SetReturnTo(BASE_URL + "/ui/login?login_challenge=" + testLoginChallenge)

	// the company sign-in returns to the login request of the app
	mocks.directory.EXPECT().SignInTenants(gomock.Any(), testRegistrationEmail).Times(1).Return([]SignInTenant{{ID: testTenant, AutoJoinCandidate: true}}, nil)
	mocks.directory.EXPECT().SignInContext(gomock.Any(), testTenant, testRegistrationEmail, "").Times(1).Return(autoJoinAdmitted(MFARequirementNone, connA), nil)
	mocks.sso.EXPECT().Options(gomock.Any(), []string{connA}).Times(1).Return([]Option{{ConnectionID: connA, Label: "Hooli"}}, nil)
	mocks.kratos.EXPECT().GetRegistrationFlow(gomock.Any(), "reg", gomock.Any()).Times(1).Return(registrationFlow, nil, nil)
	mocks.kratos.EXPECT().CreateBrowserLoginFlow(gomock.Any(), "", BASE_URL+"/ui/login?login_challenge="+testLoginChallenge, "", false, gomock.Any()).Times(1).Return(loginFlow("lf", "", "", true), nil, nil)
	mocks.expectStartAttempt(t, "lf", AttemptRequest{TenantID: testTenant, Email: testRegistrationEmail, ConnectionID: connA})
	mocks.state.EXPECT().GetStateCookie(gomock.Any()).Times(1).Return(cookies.FlowStateCookie{}, nil)
	mocks.state.EXPECT().SetStateCookie(gomock.Any(), stateFor(testTenant)).Times(1).Return(nil)

	rec := httptest.NewRecorder()
	answered := api.InterceptRegistrationSubmission(rec, registrationRequest(), "reg")

	expectAnswered(t, answered)
	expectRedirectTo(t, rec, testCompanySignIn)

	signIn, err := mocks.store.GetSignIn(requestWith(findCookie(rec, signInCookieName)))
	expectNoError(t, err)
	if signIn.LoginChallengeHash != cookies.ChallengeHash(testLoginChallenge) {
		t.Fatalf("expected the hash of the login challenge, got %s", signIn.LoginChallengeHash)
	}
}

func TestRegistrationSignInWithSeveralOptions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, mocks := newTestAPI(t, ctrl)
	context := autoJoinAdmitted(MFARequirementNone, connA, connB)
	options := []Option{{ConnectionID: connA, Label: "Hooli"}, {ConnectionID: connB, Label: "Hooli legacy"}}

	mocks.directory.EXPECT().SignInTenants(gomock.Any(), testRegistrationEmail).Times(1).Return([]SignInTenant{{ID: testTenant, AutoJoinCandidate: true}}, nil)
	mocks.directory.EXPECT().SignInContext(gomock.Any(), testTenant, testRegistrationEmail, "").Times(3).Return(context, nil)
	mocks.sso.EXPECT().Options(gomock.Any(), []string{connA, connB}).Times(2).Return(options, nil)
	mocks.expectRegistrationLoginFlow("")

	rec := httptest.NewRecorder()
	answered := api.InterceptRegistrationSubmission(rec, registrationRequest(), "reg")

	expectAnswered(t, answered)
	expectRedirectTo(t, rec, "/ui/login?flow=lf")

	// the login flow of the registration offers the company sign-ins only
	registrationCookie := findCookie(rec, registrationCookieName)
	get := getLoginFlowRequest()
	get.AddCookie(registrationCookie)
	flow, ok := api.HydrateLoginFlow(httptest.NewRecorder(), get, loginFlow("lf", "", "", true))

	expectRendered(t, ok)
	expectNodes(t, flow, ssoNodeGroup, 2)
	if hasIdentifierFirst(flow) {
		t.Fatalf("expected the identifier step to be removed")
	}
	if hasNode(flow, "sso_tenant_reset") {
		t.Fatalf("expected no way back to a tenant list")
	}

	// the pick of one of them starts it for the registration
	mocks.expectStartAttempt(t, "lf", AttemptRequest{TenantID: testTenant, Email: testRegistrationEmail, ConnectionID: connB})

	pick := jsonRequest(http.MethodPost, "/api/kratos/self-service/login?flow=lf", `{"sso_connection":"`+connB+`"}`)
	pick.AddCookie(registrationCookie)
	rec = httptest.NewRecorder()
	_, answered = api.InterceptLoginSubmission(rec, pick, loginFlow("lf", "", "", true))

	expectAnswered(t, answered)
	expectRedirectTo(t, rec, testCompanySignIn)

	signIn, err := mocks.store.GetSignIn(requestWith(findCookie(rec, signInCookieName)))
	expectNoError(t, err)
	if !signIn.Join {
		t.Fatalf("expected join to be true")
	}
	if c := findCookie(rec, registrationCookieName); c == nil || c.MaxAge >= 0 {
		t.Fatalf("expected the registration cookie to be cleared, got %v", c)
	}
}

func TestRegistrationSignInWithoutOptions(t *testing.T) {
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

			mocks.directory.EXPECT().SignInTenants(gomock.Any(), testRegistrationEmail).Times(1).Return([]SignInTenant{{ID: testTenant, AutoJoinCandidate: true}}, nil)
			mocks.directory.EXPECT().SignInContext(gomock.Any(), testTenant, testRegistrationEmail, "").Times(1).Return(autoJoinAdmitted(MFARequirementNone, connA), nil)
			mocks.sso.EXPECT().Options(gomock.Any(), []string{connA}).Times(1).Return(nil, tt.err)

			rec := httptest.NewRecorder()
			answered := api.InterceptRegistrationSubmission(rec, registrationRequest(), "reg")

			expectAnswered(t, answered)
			expectStatus(t, rec, tt.expectedStatus)
			expectErrorID(t, rec, tt.expectedErrorID)
		})
	}
}

func TestAdmittingTenants(t *testing.T) {
	tests := []struct {
		name     string
		tenant   SignInTenant
		context  *SignInContext
		expected int
	}{
		{name: "admitted by auto-join", tenant: SignInTenant{ID: testTenant, AutoJoinCandidate: true}, context: autoJoinAdmitted(MFARequirementNone, connA), expected: 1},
		{name: "account admitted by auto-join", tenant: SignInTenant{ID: testTenant, AutoJoinCandidate: true}, context: &SignInContext{AutoJoinAdmits: true, AccountExists: true, Enforcement: EnforcementRequired, ConnectionIDs: []string{connA}}, expected: 1},
		{name: "invited with no account", tenant: SignInTenant{ID: testTenant, Invited: true}, context: invited(EnforcementRequired, connA), expected: 1},
		// Kratos answers that the account exists
		{name: "invited account", tenant: SignInTenant{ID: testTenant, Invited: true}, context: &SignInContext{InvitationAdmits: true, AccountExists: true, Enforcement: EnforcementRequired, ConnectionIDs: []string{connA}}, expected: 0},
		{name: "invited to a tenant that does not require company sign-in", tenant: SignInTenant{ID: testTenant, Invited: true}, context: invited(EnforcementOptional, connA), expected: 0},
		{name: "member", tenant: SignInTenant{ID: testTenant, Invited: true}, context: member(EnforcementRequired, MFARequirementNone, connA), expected: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			api, mocks := newTestAPI(t, ctrl)

			// the tenants of the account are not asked about
			mocks.directory.EXPECT().SignInTenants(gomock.Any(), testRegistrationEmail).Times(1).Return([]SignInTenant{{ID: "personal"}, tt.tenant}, nil)
			mocks.directory.EXPECT().SignInContext(gomock.Any(), testTenant, testRegistrationEmail, "").Times(1).Return(tt.context, nil)

			admitted, err := api.admittingTenants(registrationRequest().Context(), testRegistrationEmail)

			expectNoError(t, err)
			if len(admitted) != tt.expected {
				t.Fatalf("expected %d admitting tenants, got %d", tt.expected, len(admitted))
			}
		})
	}
}

func TestAdmittingTenantsFailOnSignInContext(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, mocks := newTestAPI(t, ctrl)

	// a failure of any kind says nothing about the tenant
	mocks.directory.EXPECT().SignInTenants(gomock.Any(), testRegistrationEmail).Times(1).Return([]SignInTenant{{ID: testTenant, AutoJoinCandidate: true}}, nil)
	mocks.directory.EXPECT().SignInContext(gomock.Any(), testTenant, testRegistrationEmail, "").Times(1).Return(nil, status.Error(codes.Internal, "error"))

	admitted, err := api.admittingTenants(registrationRequest().Context(), testRegistrationEmail)

	if err == nil {
		t.Fatalf("expected error not nil")
	}
	if len(admitted) != 0 {
		t.Fatalf("expected no admitting tenant, got %d", len(admitted))
	}
}

// expectAdmittingTenants expects two tenants to admit the address of the
// registration, each time they are looked up.
func (m *apiMocks) expectAdmittingTenants(times int) {
	tenants := []SignInTenant{{ID: "a", Name: "Alpha", AutoJoinCandidate: true}, {ID: "b", Name: "Beta", Invited: true}}
	m.directory.EXPECT().SignInTenants(gomock.Any(), testRegistrationEmail).Times(times).Return(tenants, nil)
	m.directory.EXPECT().SignInContext(gomock.Any(), "a", testRegistrationEmail, "").Times(times).Return(autoJoinAdmitted(MFARequirementNone, connA), nil)
	m.directory.EXPECT().SignInContext(gomock.Any(), "b", testRegistrationEmail, "").Times(times).Return(invited(EnforcementRequired, connB), nil)
}

// registrationChoice submits a registration two tenants admit, and returns
// the registration cookie.
func registrationChoice(t *testing.T, api *API, mocks *apiMocks) *http.Cookie {
	t.Helper()
	mocks.expectRegistrationLoginFlow("")

	rec := httptest.NewRecorder()
	answered := api.InterceptRegistrationSubmission(rec, registrationRequest(), "reg")

	expectAnswered(t, answered)
	expectRedirectTo(t, rec, "/ui/login?flow=lf")
	if responseBody(t, rec)["continue_with"] == nil {
		t.Fatalf("expected continue_with to be set")
	}

	registrationCookie := findCookie(rec, registrationCookieName)
	if registrationCookie == nil {
		t.Fatalf("expected the registration cookie to be set")
	}
	return registrationCookie
}

func TestHydrateRegistrationChoiceWhenNoTenantAdmits(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, mocks := newTestAPI(t, ctrl)
	r := withCookies(getLoginFlowRequest(), func(w http.ResponseWriter) {
		_ = mocks.store.SetRegistration(w, RegistrationCookie{FlowID: "lf", Email: testRegistrationEmail})
	})

	// the tenants stopped admitting the address since the registration was submitted
	mocks.directory.EXPECT().SignInTenants(gomock.Any(), testRegistrationEmail).Times(1).Return(nil, nil)

	flow, ok := api.HydrateLoginFlow(httptest.NewRecorder(), r, loginFlow("lf", "", "", true))

	if !ok {
		t.Fatalf("expected the flow to be returned")
	}
	if !hasMessage(flow, noTenantAdmitsID) {
		t.Fatalf("expected message %d, got %v", noTenantAdmitsID, flow.Ui.Messages)
	}
}

func TestChooseRegistrationTenant(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, mocks := newTestAPI(t, ctrl)

	mocks.expectAdmittingTenants(2)
	registrationCookie := registrationChoice(t, api, mocks)

	// the pick starts the company sign-in of that tenant
	mocks.sso.EXPECT().Options(gomock.Any(), []string{connB}).Times(1).Return([]Option{{ConnectionID: connB, Label: "Beta"}}, nil)
	mocks.directory.EXPECT().SignInContext(gomock.Any(), "b", testRegistrationEmail, "").Times(1).Return(invited(EnforcementRequired, connB), nil)
	mocks.expectStartAttempt(t, "lf", AttemptRequest{TenantID: "b", Email: testRegistrationEmail, ConnectionID: connB})

	pick := jsonRequest(http.MethodPost, "/api/kratos/self-service/login?flow=lf", `{"sso_tenant":"b"}`)
	pick.AddCookie(registrationCookie)
	rec := httptest.NewRecorder()
	_, answered := api.InterceptLoginSubmission(rec, pick, loginFlow("lf", "", "", true))

	expectAnswered(t, answered)
	expectRedirectTo(t, rec, testCompanySignIn)

	signIn, err := mocks.store.GetSignIn(requestWith(findCookie(rec, signInCookieName)))
	expectNoError(t, err)
	if signIn.TenantID != "b" {
		t.Fatalf("expected tenant b, got %s", signIn.TenantID)
	}
	if !signIn.Join {
		t.Fatalf("expected join to be true")
	}
}

func TestChooseRegistrationTenantNotAdmitting(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, mocks := newTestAPI(t, ctrl)

	mocks.expectAdmittingTenants(2)
	registrationCookie := registrationChoice(t, api, mocks)

	pick := jsonRequest(http.MethodPost, "/api/kratos/self-service/login?flow=lf", `{"sso_tenant":"elsewhere"}`)
	pick.AddCookie(registrationCookie)
	rec := httptest.NewRecorder()
	_, answered := api.InterceptLoginSubmission(rec, pick, loginFlow("lf", "", "", true))

	expectAnswered(t, answered)
	expectStatus(t, rec, http.StatusForbidden)
}

func TestChooseRegistrationTenantWhenTenantServiceUnavailable(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, mocks := newTestAPI(t, ctrl)

	mocks.directory.EXPECT().SignInTenants(gomock.Any(), testRegistrationEmail).Times(1).Return([]SignInTenant{{ID: "a", AutoJoinCandidate: true}, {ID: "b", Invited: true}}, nil)
	mocks.directory.EXPECT().SignInContext(gomock.Any(), "a", testRegistrationEmail, "").Times(1).Return(nil, status.Error(codes.Unavailable, "down"))

	// the tenant is not taken for one that does not admit the address
	pick := withCookies(jsonRequest(http.MethodPost, "/api/kratos/self-service/login?flow=lf", `{"sso_tenant":"b"}`), func(w http.ResponseWriter) {
		_ = mocks.store.SetRegistration(w, RegistrationCookie{FlowID: "lf", Email: testRegistrationEmail})
	})
	rec := httptest.NewRecorder()
	_, answered := api.InterceptLoginSubmission(rec, pick, loginFlow("lf", "", "", true))

	expectAnswered(t, answered)
	expectStatus(t, rec, http.StatusServiceUnavailable)
	expectErrorMessage(t, rec, tenantsUnavailableMessage)
}

func TestRegistrationFor(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, mocks := newTestAPI(t, ctrl)
	r := withCookies(getLoginFlowRequest(), func(w http.ResponseWriter) {
		_ = mocks.store.SetRegistration(w, RegistrationCookie{FlowID: "lf", Email: testRegistrationEmail})
	})

	registration, ok := api.registrationFor(r, "lf")

	if !ok {
		t.Fatalf("expected the registration of the flow")
	}
	if registration.Email != testRegistrationEmail {
		t.Fatalf("expected email %s, got %s", testRegistrationEmail, registration.Email)
	}
	if _, ok := api.registrationFor(r, "another"); ok {
		t.Fatalf("expected no registration for another flow")
	}
	if _, ok := api.registrationFor(getLoginFlowRequest(), "lf"); ok {
		t.Fatalf("expected no registration without the cookie")
	}
}
