// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package byosso

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	kClient "github.com/ory/kratos-client-go/v25"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/canonical/identity-platform-login-ui/internal/cookies"
	"github.com/canonical/identity-platform-login-ui/pkg/kratos"
)

const (
	BASE_URL           = "https://login.example"
	testLoginChallenge = "login_challenge_2341235123231"
	testTenant         = "0190a000-0000-7000-8000-0000000000aa"
	testEmail          = "bob@acme.example"
	testAcceptRedirect = "https://app.example/callback"
	testCompanySignIn  = "https://hydra-sso.example/oauth2/auth"
)

//go:generate mockgen -build_flags=--mod=mod -package byosso -destination ./mock_logger.go -source=../../internal/logging/interfaces.go
//go:generate mockgen -build_flags=--mod=mod -package byosso -destination ./mock_interfaces.go -source=./interfaces.go
//go:generate mockgen -build_flags=--mod=mod -package byosso -destination ./mock_tracing.go -source=../../internal/tracing/interfaces.go

type apiMocks struct {
	kratos       *MockKratosServiceInterface
	sso          *MockSSOInterface
	directory    *MockDirectoryInterface
	hydra        *MockHydraInterface
	verification *MockVerificationInterface
	identities   *MockIdentityFinderInterface
	state        *MockStateCookieInterface
	tenants      *MockTenantResolverInterface
	store        *CookieStore
	logger       *MockLoggerInterface
}

// newTestAPI returns an API built on mocks, with a logger and a tracer that
// take any call. Its clock is one minute after t0.
func newTestAPI(t *testing.T, ctrl *gomock.Controller) (*API, *apiMocks) {
	t.Helper()
	mockLogger := NewMockLoggerInterface(ctrl)
	mockLogger.EXPECT().Error(gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Debugf(gomock.Any(), gomock.Any()).AnyTimes()
	mockTracer := NewMockTracingInterface(ctrl)
	mockTracer.EXPECT().Start(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
		func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
			return ctx, trace.SpanFromContext(ctx)
		},
	)

	mocks := &apiMocks{
		kratos:       NewMockKratosServiceInterface(ctrl),
		sso:          NewMockSSOInterface(ctrl),
		directory:    NewMockDirectoryInterface(ctrl),
		hydra:        NewMockHydraInterface(ctrl),
		verification: NewMockVerificationInterface(ctrl),
		identities:   NewMockIdentityFinderInterface(ctrl),
		state:        NewMockStateCookieInterface(ctrl),
		tenants:      NewMockTenantResolverInterface(ctrl),
		store:        NewCookieStore(cookies.NewEncrypt([]byte("0123456789abcdef0123456789abcdef"), mockLogger, mockTracer)),
		logger:       mockLogger,
	}

	api, err := NewAPI(
		mocks.kratos,
		mocks.sso,
		mocks.directory,
		mocks.hydra,
		mocks.verification,
		mocks.identities,
		mocks.state,
		mocks.tenants,
		mocks.store,
		BASE_URL,
		time.Hour,
		mockTracer,
		mockLogger,
	)
	expectNoError(t, err)
	api.now = func() time.Time { return t0.Add(time.Minute) }

	return api, mocks
}

// expectAccept expects the Hydra login to be accepted for tenantID.
func (m *apiMocks) expectAccept(session *kClient.Session, tenantID string) {
	redirectTo := testAcceptRedirect
	m.kratos.EXPECT().AcceptLoginRequest(gomock.Any(), session, testLoginChallenge, tenantID).Times(1).Return(
		&kratos.BrowserLocationChangeRequired{RedirectTo: &redirectTo}, nil, nil,
	)
	m.state.EXPECT().ClearStateCookie(gomock.Any()).Times(1)
}

// expectStartAttempt expects a company sign-in to start on the login flow:
// a ticket for req, submitted as the login hint of the provider, without the
// session cookie.
func (m *apiMocks) expectStartAttempt(t *testing.T, flowID string, req AttemptRequest) {
	t.Helper()
	m.sso.EXPECT().StartAttempt(gomock.Any(), req).Times(1).Return("ticket-1", nil)
	m.kratos.EXPECT().UpdateLoginFlow(gomock.Any(), flowID, gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
		func(_ context.Context, _ string, body kClient.UpdateLoginFlowBody, c []*http.Cookie) (*kratos.BrowserLocationChangeRequired, *kClient.SuccessfulNativeLogin, []*http.Cookie, error) {
			oidc := body.UpdateLoginFlowWithOidcMethod
			if oidc == nil {
				t.Fatalf("expected an oidc submission")
			}
			if oidc.Provider != providerID {
				t.Fatalf("expected provider %s, got %s", providerID, oidc.Provider)
			}
			if oidc.UpstreamParameters["login_hint"] != "ticket-1" {
				t.Fatalf("expected login_hint ticket-1, got %v", oidc.UpstreamParameters["login_hint"])
			}
			if hasSessionCookie(c) {
				t.Fatalf("expected no session cookie")
			}

			redirectTo := testCompanySignIn
			return &kratos.BrowserLocationChangeRequired{RedirectTo: &redirectTo}, nil, nil, nil
		},
	)
}

// expectNewLoginFlow expects a login flow to be created for the login
// challenge without the session cookie, and returns it.
func (m *apiMocks) expectNewLoginFlow(t *testing.T) *kClient.LoginFlow {
	t.Helper()
	flow := loginFlow("fresh", testLoginChallenge, "", true)
	m.kratos.EXPECT().CreateBrowserLoginFlow(gomock.Any(), "", BASE_URL+"/ui/login?login_challenge="+testLoginChallenge, testLoginChallenge, false, gomock.Any()).Times(1).DoAndReturn(
		func(_ context.Context, _, _, _ string, _ bool, c []*http.Cookie) (*kClient.LoginFlow, []*http.Cookie, error) {
			if hasSessionCookie(c) {
				t.Fatalf("expected no session cookie")
			}
			return flow, nil, nil
		},
	)
	return flow
}

// expectIdentify expects the identifier step of the new login flow to be
// submitted for testEmail without the session cookie, and answers redirectTo.
func (m *apiMocks) expectIdentify(t *testing.T, redirectTo *kratos.BrowserLocationChangeRequired) {
	t.Helper()
	m.kratos.EXPECT().UpdateIdentifierFirstLoginFlow(gomock.Any(), "fresh", gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
		func(_ context.Context, _ string, body kClient.UpdateLoginFlowWithIdentifierFirstMethod, c []*http.Cookie) (*kratos.BrowserLocationChangeRequired, []*http.Cookie, error) {
			if body.Identifier != testEmail {
				t.Fatalf("expected identifier %s, got %s", testEmail, body.Identifier)
			}
			if hasSessionCookie(c) {
				t.Fatalf("expected no session cookie")
			}
			return redirectTo, nil, nil
		},
	)
}

func jsonRequest(method, target, body string) *http.Request {
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, target, nil)
	} else {
		r = httptest.NewRequest(method, target, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	}
	r.Header.Set("Accept", "application/json, text/plain, */*")
	return r
}

func getLoginFlowRequest() *http.Request {
	return jsonRequest(http.MethodGet, "/api/kratos/self-service/login/flows?id=f", "")
}

func updateLoginFlowRequest(body string) *http.Request {
	return jsonRequest(http.MethodPost, "/api/kratos/self-service/login?flow=f", body)
}

func createLoginFlowRequest() *http.Request {
	return withSessionCookie(jsonRequest(http.MethodGet, "/api/kratos/self-service/login/browser?login_challenge="+testLoginChallenge, ""))
}

func withSessionCookie(r *http.Request) *http.Request {
	r.AddCookie(&http.Cookie{Name: kratos.KRATOS_SESSION_COOKIE_NAME, Value: "session"})
	return r
}

// withReceipt adds the receipt cookie sso-service left in the browser for the
// ticket.
func withReceipt(r *http.Request, ticket, receipt string) *http.Request {
	r.AddCookie(&http.Cookie{Name: receiptCookieName(ticket), Value: receipt})
	return r
}

// withCookies adds the cookies write sets to the request.
func withCookies(r *http.Request, write func(w http.ResponseWriter)) *http.Request {
	rec := httptest.NewRecorder()
	write(rec)
	for _, c := range rec.Result().Cookies() {
		r.AddCookie(c)
	}
	return r
}

// requestWith returns a request carrying the cookie the response set.
func requestWith(c *http.Cookie) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	if c != nil {
		r.AddCookie(c)
	}
	return r
}

func findCookie(rec *httptest.ResponseRecorder, name string) *http.Cookie {
	var found *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == name {
			found = c
		}
	}
	return found
}

func hasSessionCookie(c []*http.Cookie) bool {
	for _, cookie := range c {
		if cookie.Name == kratos.KRATOS_SESSION_COOKIE_NAME {
			return true
		}
	}
	return false
}

func responseBody(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected a JSON body, got %q", rec.Body.String())
	}
	return body
}

func expectNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("expected error to be nil, got %v", err)
	}
}

// expectAnswered checks that a hook answered the request itself.
func expectAnswered(t *testing.T, answered bool) {
	t.Helper()
	if !answered {
		t.Fatalf("expected the request to be answered")
	}
}

// expectRendered checks that a hook left the flow to be rendered.
func expectRendered(t *testing.T, ok bool) {
	t.Helper()
	if !ok {
		t.Fatalf("expected the flow to be rendered")
	}
}

func expectStatus(t *testing.T, rec *httptest.ResponseRecorder, expected int) {
	t.Helper()
	if rec.Code != expected {
		t.Fatalf("expected HTTP status code %d, got %d", expected, rec.Code)
	}
}

func expectReceiptCleared(t *testing.T, rec *httptest.ResponseRecorder, ticket string) {
	t.Helper()
	if c := findCookie(rec, receiptCookieName(ticket)); c == nil || c.MaxAge >= 0 {
		t.Fatalf("expected the receipt cookie of %s to be cleared, got %v", ticket, c)
	}
}

func expectRedirectTo(t *testing.T, rec *httptest.ResponseRecorder, expected string) {
	t.Helper()
	if redirectTo, _ := responseBody(t, rec)["redirect_to"].(string); redirectTo != expected {
		t.Fatalf("expected redirect to %s, got %s", expected, redirectTo)
	}
}

func expectErrorID(t *testing.T, rec *httptest.ResponseRecorder, expected string) {
	t.Helper()
	e, _ := responseBody(t, rec)["error"].(map[string]any)
	if id, _ := e["id"].(string); id != expected {
		t.Fatalf("expected error id %s, got %s", expected, id)
	}
}

func expectErrorMessage(t *testing.T, rec *httptest.ResponseRecorder, expected string) {
	t.Helper()
	e, _ := responseBody(t, rec)["error"].(map[string]any)
	if message, _ := e["message"].(string); message != expected {
		t.Fatalf("expected error message %q, got %q", expected, message)
	}
}

// expectNodes checks how many nodes of group the flow has.
func expectNodes(t *testing.T, flow *kClient.LoginFlow, group string, expected int) {
	t.Helper()
	if nodes := len(nodesOf(flow, group)); nodes != expected {
		t.Fatalf("expected %d %s nodes, got %d", expected, group, nodes)
	}
}

func stateFor(tenantID string) cookies.FlowStateCookie {
	return cookies.FlowStateCookie{LoginChallengeHash: cookies.ChallengeHash(testLoginChallenge), TenantID: tenantID}
}

func chosenStateFor(tenantID string) cookies.FlowStateCookie {
	stateCookie := stateFor(tenantID)
	stateCookie.TenantChoice = true
	return stateCookie
}

func inputNode(group, name, value string) kClient.UiNode {
	attrs := kClient.NewUiNodeInputAttributesWithDefaults()
	attrs.Name = name
	attrs.Value = value
	attrs.NodeType = "input"

	node := kClient.NewUiNodeWithDefaults()
	node.Type = "input"
	node.Group = group
	node.Attributes = kClient.UiNodeAttributes{UiNodeInputAttributes: attrs}
	return *node
}

// loginFlow builds a login flow with the identifier. At the identifier step
// it has the submit of that step, past it the password and a public sign-in.
func loginFlow(id, loginChallenge, identifier string, identifierStep bool) *kClient.LoginFlow {
	flow := kClient.NewLoginFlowWithDefaults()
	flow.Id = id
	flow.IssuedAt = t0
	if loginChallenge != "" {
		flow.Oauth2LoginChallenge = &loginChallenge
		requestURL := "https://hydra.example/oauth2/auth?client_id=app"
		flow.Oauth2LoginRequest = &kClient.OAuth2LoginRequest{RequestUrl: &requestURL}
	}
	flow.Ui.Nodes = []kClient.UiNode{inputNode("default", "csrf_token", "csrf-1"), inputNode("default", "identifier", identifier)}
	if identifierStep {
		flow.Ui.Nodes = append(flow.Ui.Nodes, inputNode("identifier_first", "method", "identifier_first"))
	} else {
		flow.Ui.Nodes = append(flow.Ui.Nodes, inputNode("password", "password", ""), inputNode("oidc", "provider", "google"))
	}
	return flow
}

// refusedLoginFlow builds a login flow Kratos refused at the identifier step.
func refusedLoginFlow() *kClient.LoginFlow {
	flow := loginFlow("f", testLoginChallenge, testEmail, true)
	flow.Ui.Messages = append(flow.Ui.Messages, *kClient.NewUiText(kratos.IncorrectAccountIdentifier, "does not exist", "error"))
	return flow
}

func nodesOf(flow *kClient.LoginFlow, group string) []kClient.UiNode {
	var nodes []kClient.UiNode
	for _, node := range flow.Ui.Nodes {
		if node.Group == group {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

func hasNode(flow *kClient.LoginFlow, name string) bool {
	for _, node := range flow.Ui.Nodes {
		if attrs := node.Attributes.UiNodeInputAttributes; attrs != nil && attrs.Name == name {
			return true
		}
	}
	return false
}

func hasMessage(flow *kClient.LoginFlow, id int64) bool {
	for _, message := range flow.Ui.Messages {
		if message.Id == id {
			return true
		}
	}
	return false
}

func settingsFlow() *kClient.SettingsFlow {
	flow := kClient.NewSettingsFlowWithDefaults()
	flow.Id = "sf"
	flow.Identity = *kClient.NewIdentity("iid", "default", "", map[string]interface{}{"email": testEmail})
	return flow
}

// kratosError builds the error the kratos service returns for an error
// response of Kratos that carries an ID.
func kratosError(id string, code int64) error {
	return &kratos.KratosGenericError{Response: kratos.KratosErrorResponse{Error: &kClient.GenericError{Id: &id, Code: &code}}}
}

func tenantServiceUnavailable() error {
	return fmt.Errorf("cannot lookup tenants by email: %w", status.Error(codes.Unavailable, "connection refused"))
}

func TestHydrateLoginFlowUnchanged(t *testing.T) {
	tests := []struct {
		name       string
		flow       func() *kClient.LoginFlow
		setupMocks func(mocks *apiMocks)
	}{
		{
			name: "aal2 flow",
			flow: func() *kClient.LoginFlow {
				flow := loginFlow("f", testLoginChallenge, testEmail, false)
				flow.SetRequestedAal(kClient.AUTHENTICATORASSURANCELEVEL_AAL2)
				return flow
			},
		},
		{
			name: "flow without login challenge",
			flow: func() *kClient.LoginFlow { return loginFlow("f", "", testEmail, false) },
		},
		{
			name: "state cookie fails",
			flow: func() *kClient.LoginFlow { return loginFlow("f", testLoginChallenge, testEmail, false) },
			setupMocks: func(mocks *apiMocks) {
				mocks.state.EXPECT().GetStateCookie(gomock.Any()).Times(1).Return(cookies.FlowStateCookie{}, errors.New("error"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			api, mocks := newTestAPI(t, ctrl)
			flow := tt.flow()

			if tt.setupMocks != nil {
				tt.setupMocks(mocks)
			}

			hydrated, ok := api.HydrateLoginFlow(httptest.NewRecorder(), getLoginFlowRequest(), flow)

			expectRendered(t, ok)
			if hydrated != flow {
				t.Fatalf("expected the flow to be unchanged")
			}
			if len(hydrated.Ui.Nodes) != 4 {
				t.Fatalf("expected 4 nodes, got %d", len(hydrated.Ui.Nodes))
			}
		})
	}
}

func TestBeforeTenantSelectionWithTenants(t *testing.T) {
	tests := []struct {
		name       string
		tenants    []SignInTenant
		setupMocks func(mocks *apiMocks)
	}{
		{
			name:    "several tenants",
			tenants: []SignInTenant{{ID: "personal"}, {ID: testTenant}},
			setupMocks: func(mocks *apiMocks) {
				mocks.state.EXPECT().SetStateCookie(gomock.Any(), cookies.FlowStateCookie{TenantChoice: true}).Times(1).Return(nil)
			},
		},
		{
			name:    "one tenant",
			tenants: []SignInTenant{{ID: testTenant}},
			setupMocks: func(mocks *apiMocks) {
				mocks.directory.EXPECT().SignInContext(gomock.Any(), testTenant, testEmail, "").Times(1).Return(member(EnforcementOptional, MFARequirementNone, connA), nil)
				// a choice left by an earlier sign-in does not make this tenant a chosen one
				mocks.state.EXPECT().GetStateCookie(gomock.Any()).Times(1).Return(cookies.FlowStateCookie{TenantChoice: true}, nil)
				mocks.tenants.EXPECT().TenantID(cookies.FlowStateCookie{TenantChoice: true}, testLoginChallenge).Times(1).Return("")
				mocks.state.EXPECT().SetStateCookie(gomock.Any(), stateFor(testTenant)).Times(1).Return(nil)
			},
		},
		{
			name:    "no tenant",
			tenants: nil,
			setupMocks: func(mocks *apiMocks) {
				mocks.state.EXPECT().GetStateCookie(gomock.Any()).Times(1).Return(cookies.FlowStateCookie{}, nil)
				mocks.state.EXPECT().SetStateCookie(gomock.Any(), stateFor(cookies.NoTenantAvailable)).Times(1).Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			api, mocks := newTestAPI(t, ctrl)
			flow := loginFlow("f", testLoginChallenge, testEmail, false)

			mocks.directory.EXPECT().SignInTenants(underBudget{}, testEmail).Times(1).Return(tt.tenants, nil)
			tt.setupMocks(mocks)

			rec := httptest.NewRecorder()
			answered := api.BeforeTenantSelection(rec, updateLoginFlowRequest(`{}`), flow, testEmail)

			expectAnswered(t, answered)
			expectRedirectTo(t, rec, "/ui/login?flow=f")
		})
	}
}

func TestBeforeTenantSelectionWithoutEmailOrLoginChallenge(t *testing.T) {
	tests := []struct {
		name  string
		flow  *kClient.LoginFlow
		email string
	}{
		{name: "no email", flow: loginFlow("f", testLoginChallenge, "", true), email: ""},
		{name: "no login challenge", flow: loginFlow("f", "", testEmail, false), email: testEmail},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			api, _ := newTestAPI(t, ctrl)

			if api.BeforeTenantSelection(httptest.NewRecorder(), updateLoginFlowRequest(`{}`), tt.flow, tt.email) {
				t.Fatalf("expected the request not to be answered")
			}
		})
	}
}

func TestBeforeTenantSelectionWhenTenantServiceUnavailable(t *testing.T) {
	tests := []struct {
		name string
		flow *kClient.LoginFlow
	}{
		{name: "address of an account", flow: loginFlow("f", testLoginChallenge, testEmail, false)},
		{name: "address Kratos refused", flow: refusedLoginFlow()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			api, mocks := newTestAPI(t, ctrl)

			mocks.directory.EXPECT().SignInTenants(gomock.Any(), testEmail).Times(1).Return(nil, tenantServiceUnavailable())

			rec := httptest.NewRecorder()
			answered := api.BeforeTenantSelection(rec, updateLoginFlowRequest(`{}`), tt.flow, testEmail)

			expectAnswered(t, answered)
			expectStatus(t, rec, http.StatusServiceUnavailable)
			expectErrorID(t, rec, ssoUnavailableError)
			expectErrorMessage(t, rec, tenantsUnavailableMessage)
		})
	}
}

func TestBeforeTenantSelectionFailOnSignInTenants(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, mocks := newTestAPI(t, ctrl)
	flow := loginFlow("f", testLoginChallenge, testEmail, false)
	body := `{"method":"identifier_first","identifier":"bob@acme.example"}`
	r := updateLoginFlowRequest(body)

	mocks.directory.EXPECT().SignInTenants(gomock.Any(), testEmail).Times(1).Return(nil, errors.New("error"))

	rec := httptest.NewRecorder()
	answered := api.BeforeTenantSelection(rec, r, flow, testEmail)

	if answered {
		t.Fatalf("expected the request not to be answered")
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("expected an empty body, got %q", rec.Body.String())
	}
	if _, ok := r.Context().Deadline(); ok {
		t.Fatalf("expected the request of the handler to have no deadline")
	}
	if raw, _ := io.ReadAll(r.Body); string(raw) != body {
		t.Fatalf("expected body %q, got %q", body, raw)
	}
}

func TestBeforeTenantSelectionFailOnSignInTenantsOfRefusedAddress(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("no account holds the address", func(t *testing.T) {
		api, mocks := newTestAPI(t, ctrl)

		// tenant-service refuses an identifier that is not an address
		mocks.directory.EXPECT().SignInTenants(gomock.Any(), testEmail).Times(1).Return(nil, status.Error(codes.InvalidArgument, "invalid email"))
		mocks.identities.EXPECT().IdentityExists(gomock.Any(), testEmail).Times(1).Return(false, nil)
		mocks.state.EXPECT().GetStateCookie(gomock.Any()).Times(1).Return(cookies.FlowStateCookie{}, nil)
		mocks.state.EXPECT().SetStateCookie(gomock.Any(), stateFor(cookies.NoTenantAvailable)).Times(1).Return(nil)

		rec := httptest.NewRecorder()
		answered := api.BeforeTenantSelection(rec, updateLoginFlowRequest(`{}`), refusedLoginFlow(), testEmail)

		expectAnswered(t, answered)
		expectRedirectTo(t, rec, "/ui/login?flow=f")
	})

	t.Run("an account holds the address", func(t *testing.T) {
		api, mocks := newTestAPI(t, ctrl)

		mocks.directory.EXPECT().SignInTenants(gomock.Any(), testEmail).Times(1).Return(nil, status.Error(codes.InvalidArgument, "invalid email"))
		mocks.identities.EXPECT().IdentityExists(gomock.Any(), testEmail).Times(1).Return(true, nil)

		if api.BeforeTenantSelection(httptest.NewRecorder(), updateLoginFlowRequest(`{}`), refusedLoginFlow(), testEmail) {
			t.Fatalf("expected the request not to be answered")
		}
	})
}

func TestInterceptLoginSubmissionWithProvider(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		expectedStatus int
	}{
		{name: "with the oidc method", body: `{"method":"oidc","provider":"byo-sso"}`, expectedStatus: http.StatusForbidden},
		{name: "without method", body: `{"provider":"byo-sso"}`, expectedStatus: http.StatusForbidden},
		// the handler decodes the body into a struct, which takes these too
		{name: "field in another case", body: `{"method":"oidc","Provider":"byo-sso"}`, expectedStatus: http.StatusForbidden},
		{name: "field named twice", body: `{"provider":"google","Provider":"byo-sso"}`, expectedStatus: http.StatusBadRequest},
		{name: "object followed by more", body: `{"method":"oidc","provider":"byo-sso","identifier":"x"} {}`, expectedStatus: http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			api, mocks := newTestAPI(t, ctrl)

			// the page never submits this: a warning
			mocks.logger.EXPECT().Warnf(gomock.Any(), gomock.Any()).Times(1)

			rec := httptest.NewRecorder()
			_, answered := api.InterceptLoginSubmission(rec, updateLoginFlowRequest(tt.body), loginFlow("f", testLoginChallenge, testEmail, false))

			expectAnswered(t, answered)
			expectStatus(t, rec, tt.expectedStatus)
		})
	}
}

func TestInterceptLoginSubmissionWithFirstFactor(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, mocks := newTestAPI(t, ctrl)
	body := `{"method":"password","identifier":"bob@acme.example","password":"test"}`
	r := updateLoginFlowRequest(body)

	rec := httptest.NewRecorder()
	next, answered := api.InterceptLoginSubmission(rec, r, loginFlow("f", testLoginChallenge, testEmail, false))

	if answered {
		t.Fatalf("expected the request not to be answered")
	}
	if _, ok := next.Context().Value(freshKey{}).(FreshCookie); !ok {
		t.Fatalf("expected the returned request to carry the fresh mark")
	}
	if _, ok := r.Context().Value(freshKey{}).(FreshCookie); ok {
		t.Fatalf("expected the given request to be unchanged")
	}
	if _, ok := next.Context().Deadline(); ok {
		t.Fatalf("expected the returned request to have no deadline")
	}
	if raw, _ := io.ReadAll(next.Body); string(raw) != body {
		t.Fatalf("expected body %q, got %q", body, raw)
	}

	mark, err := mocks.store.GetFresh(requestWith(findCookie(rec, freshCookieName)))
	expectNoError(t, err)
	if mark.LoginChallengeHash != cookies.ChallengeHash(testLoginChallenge) {
		t.Fatalf("expected the fresh cookie of the login challenge, got %+v", mark)
	}
}

func TestInterceptLoginSubmissionWithSecondFactor(t *testing.T) {
	tests := []struct {
		name string
		flow func() *kClient.LoginFlow
		body string
	}{
		{
			name: "aal2 flow",
			flow: func() *kClient.LoginFlow {
				flow := loginFlow("f", testLoginChallenge, testEmail, false)
				flow.SetRequestedAal(kClient.AUTHENTICATORASSURANCELEVEL_AAL2)
				return flow
			},
			body: `{"method":"totp","totp_code":"123456"}`,
		},
		{
			name: "flow without login challenge",
			flow: func() *kClient.LoginFlow { return loginFlow("f", "", testEmail, false) },
			body: `{"method":"password","password":"test"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			api, _ := newTestAPI(t, ctrl)
			r := updateLoginFlowRequest(tt.body)

			rec := httptest.NewRecorder()
			next, answered := api.InterceptLoginSubmission(rec, r, tt.flow())

			if answered {
				t.Fatalf("expected the request not to be answered")
			}
			if next != r {
				t.Fatalf("expected the request to be unchanged")
			}
			if c := findCookie(rec, freshCookieName); c != nil {
				t.Fatalf("expected no fresh cookie, got %v", c)
			}
		})
	}
}

func TestBeforeAcceptLoginWithoutLoginChallengeOrSession(t *testing.T) {
	tests := []struct {
		name           string
		session        *kClient.Session
		loginChallenge string
	}{
		{name: "no login challenge", session: passwordSession("s1"), loginChallenge: ""},
		{name: "no session", session: nil, loginChallenge: testLoginChallenge},
		{name: "session without identity", session: kClient.NewSessionWithDefaults(), loginChallenge: testLoginChallenge},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			api, _ := newTestAPI(t, ctrl)

			if api.BeforeAcceptLogin(httptest.NewRecorder(), createLoginFlowRequest(), tt.session, tt.loginChallenge, stateFor(testTenant)) {
				t.Fatalf("expected the request not to be answered")
			}
		})
	}
}

func TestBeforeAcceptLogin(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, mocks := newTestAPI(t, ctrl)
	session := passwordSession("s1", totp())

	mocks.tenants.EXPECT().TenantID(stateFor(testTenant), testLoginChallenge).AnyTimes().Return(testTenant)
	mocks.hydra.EXPECT().LoginRequest(underBudget{}, testLoginChallenge).Times(1).Return(&LoginRequest{}, nil)
	mocks.directory.EXPECT().SignInContext(gomock.Any(), testTenant, "", "iid").Times(1).Return(member(EnforcementOff, MFARequirementNone), nil)
	mocks.expectAccept(session, testTenant)

	rec := httptest.NewRecorder()
	answered := api.BeforeAcceptLogin(rec, createLoginFlowRequest(), session, testLoginChallenge, stateFor(testTenant))

	expectAnswered(t, answered)
	expectRedirectTo(t, rec, testAcceptRedirect)
}

func TestInterceptRegistrationSubmissionRefused(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		expectedStatus int
	}{
		{name: "company sign-in provider", body: `{"method":"oidc","provider":"byo-sso"}`, expectedStatus: http.StatusForbidden},
		// the handler reads the flat address for the profile method, the nested one otherwise
		{name: "two addresses", body: `{"method":"profile","traits":{"email":"bob@acme.example"},"traits.email":"hank@hooli.example"}`, expectedStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			api, mocks := newTestAPI(t, ctrl)

			mocks.logger.EXPECT().Warnf(gomock.Any(), gomock.Any()).Times(1)

			rec := httptest.NewRecorder()
			answered := api.InterceptRegistrationSubmission(rec, jsonRequest(http.MethodPost, "/api/kratos/self-service/registration?flow=reg", tt.body), "reg")

			expectAnswered(t, answered)
			expectStatus(t, rec, tt.expectedStatus)
		})
	}
}

func TestInterceptRegistrationSubmissionWithoutEmail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, _ := newTestAPI(t, ctrl)

	if api.InterceptRegistrationSubmission(httptest.NewRecorder(), jsonRequest(http.MethodPost, "/api/kratos/self-service/registration?flow=reg", `{"method":"oidc","provider":"google"}`), "reg") {
		t.Fatalf("expected the request not to be answered")
	}
}

func TestInterceptRegistrationSubmissionWhenNoTenantAdmits(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, mocks := newTestAPI(t, ctrl)
	body := `{"method":"password","traits":{"email":"bob@acme.example"}}`
	r := jsonRequest(http.MethodPost, "/api/kratos/self-service/registration?flow=reg", body)

	mocks.directory.EXPECT().SignInTenants(underBudget{}, testEmail).Times(1).Return(nil, nil)

	if api.InterceptRegistrationSubmission(httptest.NewRecorder(), r, "reg") {
		t.Fatalf("expected the request not to be answered")
	}
	if _, ok := r.Context().Deadline(); ok {
		t.Fatalf("expected the request of the handler to have no deadline")
	}
	if raw, _ := io.ReadAll(r.Body); string(raw) != body {
		t.Fatalf("expected body %q, got %q", body, raw)
	}
}

func TestInterceptRegistrationSubmissionFailOnSignInTenants(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, mocks := newTestAPI(t, ctrl)
	r := jsonRequest(http.MethodPost, "/api/kratos/self-service/registration?flow=reg", `{"method":"password","traits.email":"bob@acme.example"}`)

	mocks.directory.EXPECT().SignInTenants(gomock.Any(), testEmail).Times(1).Return(nil, errors.New("error"))

	// The address may be one that tenants admit: it does not go on to Kratos
	// as an ordinary registration.
	rec := httptest.NewRecorder()
	answered := api.InterceptRegistrationSubmission(rec, r, "reg")

	expectAnswered(t, answered)
	expectStatus(t, rec, http.StatusServiceUnavailable)
}

func TestInterceptRegistrationSubmissionWithMalformedAddress(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, mocks := newTestAPI(t, ctrl)
	r := jsonRequest(http.MethodPost, "/api/kratos/self-service/registration?flow=reg", `{"method":"password","traits":{"email":"bob"}}`)

	mocks.directory.EXPECT().SignInTenants(gomock.Any(), "bob").Times(1).Return(nil, status.Error(codes.InvalidArgument, "invalid email"))

	// Kratos says what is wrong with the address
	if api.InterceptRegistrationSubmission(httptest.NewRecorder(), r, "reg") {
		t.Fatalf("expected the request not to be answered")
	}
}

func TestHydrateSettingsFlowFailOnCheckSession(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, mocks := newTestAPI(t, ctrl)
	flow := kClient.NewSettingsFlowWithDefaults()

	mocks.kratos.EXPECT().CheckSession(gomock.Any(), gomock.Any()).Times(1).Return(nil, nil, errors.New("error"))

	hydrated := api.HydrateSettingsFlow(context.Background(), flow, nil)

	if len(hydrated.Ui.Nodes) != 0 {
		t.Fatalf("expected no nodes, got %d", len(hydrated.Ui.Nodes))
	}
}

func TestHydrateSettingsFlowFailOnLinks(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, mocks := newTestAPI(t, ctrl)

	mocks.sso.EXPECT().Links(underBudget{}, "iid").Times(1).Return(nil, errors.New("error"))

	flow := api.HydrateSettingsFlow(context.Background(), settingsFlow(), nil)

	if len(flow.Ui.Nodes) != 0 {
		t.Fatalf("expected no nodes, got %d", len(flow.Ui.Nodes))
	}
}

func TestInterceptSettingsSubmissionWithProvider(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "link", body: `{"method":"oidc","link":"byo-sso"}`},
		{name: "unlink", body: `{"method":"oidc","unlink":"byo-sso"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			api, mocks := newTestAPI(t, ctrl)

			mocks.logger.EXPECT().Warnf(gomock.Any(), gomock.Any()).Times(1)

			rec := httptest.NewRecorder()
			answered := api.InterceptSettingsSubmission(rec, jsonRequest(http.MethodPost, "/api/kratos/self-service/settings?flow=sf", tt.body), "sf")

			expectAnswered(t, answered)
			expectStatus(t, rec, http.StatusForbidden)
		})
	}
}

func TestInterceptSettingsSubmissionWithOtherMethod(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "totp code", body: `{"method":"totp","totp_code":"123456"}`},
		{name: "totp unlink", body: `{"method":"totp","totp_unlink":true}`},
		{name: "another provider", body: `{"method":"oidc","unlink":"google"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			api, _ := newTestAPI(t, ctrl)
			r := withSessionCookie(jsonRequest(http.MethodPost, "/api/kratos/self-service/settings?flow=sf", tt.body))

			if api.InterceptSettingsSubmission(httptest.NewRecorder(), r, "sf") {
				t.Fatalf("expected the request not to be answered")
			}
		})
	}
}

func TestInterceptSettingsSubmissionWithUnlink(t *testing.T) {
	tests := []struct {
		name           string
		header         string
		value          string
		setupMocks     func(mocks *apiMocks)
		expectedStatus int
	}{
		{
			// a request from another site is refused with a warning
			name:   "another origin",
			header: "Origin",
			value:  "https://evil.example",
			setupMocks: func(mocks *apiMocks) {
				mocks.logger.EXPECT().Warnf(gomock.Any(), gomock.Any()).Times(1)
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:   "cross-site fetch",
			header: "Sec-Fetch-Site",
			value:  "cross-site",
			setupMocks: func(mocks *apiMocks) {
				mocks.logger.EXPECT().Warnf(gomock.Any(), gomock.Any()).Times(1)
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			// a browser always sends one of the two
			name: "neither header",
			setupMocks: func(mocks *apiMocks) {
				mocks.logger.EXPECT().Warnf(gomock.Any(), gomock.Any()).Times(1)
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			// the unlink goes on, and needs a session
			name:   "origin of login-ui",
			header: "Origin",
			value:  BASE_URL,
			setupMocks: func(mocks *apiMocks) {
				mocks.kratos.EXPECT().CheckSession(underBudget{}, gomock.Any()).Times(1).Return(nil, nil, errors.New("no session"))
			},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			api, mocks := newTestAPI(t, ctrl)
			r := unlinkRequest()
			r.Header.Del("Origin")
			if tt.header != "" {
				r.Header.Set(tt.header, tt.value)
			}

			if tt.setupMocks != nil {
				tt.setupMocks(mocks)
			}

			rec := httptest.NewRecorder()
			answered := api.InterceptSettingsSubmission(rec, r, "sf")

			expectAnswered(t, answered)
			expectStatus(t, rec, tt.expectedStatus)
		})
	}
}

func completeRequest(mocks *apiMocks, returnTo string, signIn SignInCookie) *http.Request {
	r := withSessionCookie(httptest.NewRequest(http.MethodGet, completePath+"?return_to="+returnTo, nil))
	return withCookies(r, func(w http.ResponseWriter) {
		_ = mocks.store.SetSignIn(w, signIn)
	})
}

func TestHandleCompleteWithoutSession(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, mocks := newTestAPI(t, ctrl)

	mocks.kratos.EXPECT().CheckSession(underBudget{}, gomock.Any()).Times(1).Return(nil, nil, errors.New("no session"))

	rec := httptest.NewRecorder()
	mux := chi.NewMux()
	api.RegisterEndpoints(mux)

	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v0/sso/complete", nil))

	expectStatus(t, rec, http.StatusSeeOther)
	if location := rec.Header().Get("Location"); location != BASE_URL+"/ui/login" {
		t.Fatalf("expected location %s/ui/login, got %s", BASE_URL, location)
	}
}

func TestHandleComplete(t *testing.T) {
	tests := []struct {
		name             string
		returnTo         string
		expectedLocation string
	}{
		{name: "return_to of the attempt", returnTo: "https://portal.example/welcome", expectedLocation: "https://portal.example/welcome"},
		{name: "another return_to", returnTo: "https://evil.example/", expectedLocation: BASE_URL + "/ui/manage_details"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			api, mocks := newTestAPI(t, ctrl)
			session := companySession("s-new")
			r := completeRequest(mocks, tt.returnTo, SignInCookie{Ticket: "ticket", ReturnTo: "https://portal.example/welcome"})

			mocks.kratos.EXPECT().CheckSession(gomock.Any(), gomock.Any()).Times(1).Return(session, nil, nil)
			mocks.sso.EXPECT().CompleteAttempt(gomock.Any(), "ticket", "iid", "").Times(1).Return(&Completion{ConnectionID: connA, TenantID: testTenant}, nil)

			rec := httptest.NewRecorder()
			api.handleComplete(rec, r)

			expectStatus(t, rec, http.StatusSeeOther)
			if location := rec.Header().Get("Location"); location != tt.expectedLocation {
				t.Fatalf("expected location %s, got %s", tt.expectedLocation, location)
			}

			provenance, err := mocks.store.GetProvenance(requestWith(findCookie(rec, provenanceCookieName)))
			expectNoError(t, err)
			if provenance.ConnectionID != connA {
				t.Fatalf("expected provenance of connection %s, got %+v", connA, provenance)
			}
		})
	}
}

func TestHandleCompleteJoinsTenant(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, mocks := newTestAPI(t, ctrl)
	session := companySession("s-new")
	r := completeRequest(mocks, "", SignInCookie{Ticket: "ticket", Join: true})

	mocks.kratos.EXPECT().CheckSession(gomock.Any(), gomock.Any()).Times(1).Return(session, nil, nil)
	mocks.sso.EXPECT().CompleteAttempt(gomock.Any(), "ticket", "iid", "").Times(1).Return(&Completion{ConnectionID: connA, TenantID: testTenant}, nil)
	mocks.directory.EXPECT().JoinTenant(underBudget{}, testTenant, "iid").Times(1).Return(nil)

	rec := httptest.NewRecorder()
	api.handleComplete(rec, r)

	expectStatus(t, rec, http.StatusSeeOther)
	if location := rec.Header().Get("Location"); location != BASE_URL+"/ui/manage_details" {
		t.Fatalf("expected location %s/ui/manage_details, got %s", BASE_URL, location)
	}
}

func TestHandleCompleteWithUnverifiedAddress(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, mocks := newTestAPI(t, ctrl)
	r := completeRequest(mocks, "", SignInCookie{Ticket: "ticket", Join: true})

	mocks.kratos.EXPECT().CheckSession(gomock.Any(), gomock.Any()).Times(1).Return(unverifiedCompanySession(), nil, nil)
	mocks.sso.EXPECT().CompleteAttempt(gomock.Any(), "ticket", "iid", "").Times(1).Return(&Completion{ConnectionID: connA, TenantID: testTenant}, nil)
	// The account joins once its address is verified, at a sign-in to the tenant.
	mocks.directory.EXPECT().JoinTenant(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	rec := httptest.NewRecorder()
	api.handleComplete(rec, r)

	expectStatus(t, rec, http.StatusSeeOther)
}

func TestHandleCompleteFailOnJoinTenant(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "not admitted",
			err:            fmt.Errorf("cannot join tenant: %w", errNotAdmitted),
			expectedStatus: http.StatusForbidden,
			expectedBody:   "This account is not a member of that tenant",
		},
		{
			name:           "tenant-service unavailable",
			err:            tenantServiceUnavailable(),
			expectedStatus: http.StatusServiceUnavailable,
			expectedBody:   ssoUnavailableMessage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			api, mocks := newTestAPI(t, ctrl)
			session := companySession("s-new")
			r := completeRequest(mocks, "", SignInCookie{Ticket: "ticket", Join: true})

			mocks.kratos.EXPECT().CheckSession(gomock.Any(), gomock.Any()).Times(1).Return(session, nil, nil)
			mocks.sso.EXPECT().CompleteAttempt(gomock.Any(), "ticket", "iid", "").Times(1).Return(&Completion{ConnectionID: connA, TenantID: testTenant}, nil)
			mocks.directory.EXPECT().JoinTenant(gomock.Any(), testTenant, "iid").Times(1).Return(tt.err)

			rec := httptest.NewRecorder()
			api.handleComplete(rec, r)

			expectStatus(t, rec, tt.expectedStatus)
			if body := strings.TrimSpace(rec.Body.String()); body != tt.expectedBody {
				t.Fatalf("expected body %q, got %q", tt.expectedBody, body)
			}
			// the account stays, and so does its session
			if c := findCookie(rec, kratos.KRATOS_SESSION_COOKIE_NAME); c != nil {
				t.Fatalf("expected the session cookie to be kept, got %v", c)
			}
		})
	}
}

func TestHandleCompleteFailOnCompleteAttempt(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, mocks := newTestAPI(t, ctrl)
	session := companySession("s-new")
	r := completeRequest(mocks, "", SignInCookie{Ticket: "ticket", Join: true})

	mocks.kratos.EXPECT().CheckSession(gomock.Any(), gomock.Any()).Times(1).Return(session, nil, nil)
	mocks.sso.EXPECT().CompleteAttempt(gomock.Any(), "ticket", "iid", "").Times(1).Return(nil, errors.New("error"))

	rec := httptest.NewRecorder()
	api.handleComplete(rec, r)

	expectStatus(t, rec, http.StatusServiceUnavailable)
}
