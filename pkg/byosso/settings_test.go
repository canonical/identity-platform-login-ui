// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package byosso

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	kClient "github.com/ory/kratos-client-go/v25"
	"go.uber.org/mock/gomock"

	"github.com/canonical/identity-platform-login-ui/pkg/kratos"
)

const testRefreshReturnTo = BASE_URL + "/ui/manage_connected_accounts"

// unlinkRequest builds the request the Connected accounts page sends.
func unlinkRequest() *http.Request {
	r := withSessionCookie(jsonRequest(http.MethodPost, "/api/kratos/self-service/settings?flow=sf", fmt.Sprintf(`{"sso_unlink":%q}`, connA)))
	r.Header.Set("Origin", BASE_URL)
	return r
}

// privilegedSession builds a session that authenticated within the
// privileged session age.
func privilegedSession() *kClient.Session {
	session := passwordSession("s1")
	authenticatedAt := t0.Add(59 * time.Minute)
	session.AuthenticatedAt = &authenticatedAt
	return session
}

// refreshFlow builds the refresh login of Kratos for the session of the
// browser, with no Hydra login request.
func refreshFlow() *kClient.LoginFlow {
	flow := kClient.NewLoginFlowWithDefaults()
	flow.Id = "refresh"
	flow.SetRefresh(true)
	flow.SetReturnTo(testRefreshReturnTo)
	flow.Ui.Nodes = []kClient.UiNode{inputNode("default", "csrf_token", "csrf-1"), inputNode("default", "identifier", testEmail)}
	return flow
}

func refreshPickRequest(connectionID string) *http.Request {
	return jsonRequest(http.MethodPost, "/api/kratos/self-service/login?flow=refresh", `{"sso_reauthenticate":"`+connectionID+`"}`)
}

func TestUnlinkRefused(t *testing.T) {
	tests := []struct {
		name            string
		err             error
		expectedMessage string
	}{
		{name: "last credential", err: fmt.Errorf("cannot remove the company sign-in: %w", errLastCredential), expectedMessage: "You cannot remove your only way to sign in. Set a password first."},
		{name: "no link", err: fmt.Errorf("cannot remove the company sign-in: %w", errNoLink), expectedMessage: "This account is not linked to that sign-in method."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			api, mocks := newTestAPI(t, ctrl)

			mocks.kratos.EXPECT().CheckSession(gomock.Any(), gomock.Any()).Times(1).Return(privilegedSession(), nil, nil)
			mocks.kratos.EXPECT().GetSettingsFlow(gomock.Any(), "sf", gomock.Any()).Times(2).Return(settingsFlow(), nil, nil)
			mocks.sso.EXPECT().DeleteLink(gomock.Any(), connA, "iid").Times(1).Return(tt.err)
			mocks.sso.EXPECT().Links(gomock.Any(), "iid").Times(1).Return([]Link{{ConnectionID: connA, Label: "Acme"}}, nil)

			rec := httptest.NewRecorder()
			api.InterceptSettingsSubmission(rec, unlinkRequest(), "sf")

			expectStatus(t, rec, http.StatusOK)
			if !strings.Contains(rec.Body.String(), tt.expectedMessage) {
				t.Fatalf("expected message %q, got %q", tt.expectedMessage, rec.Body.String())
			}
			// the flow still lists the company sign-in
			if !strings.Contains(rec.Body.String(), "sso_link_"+connA) {
				t.Fatalf("expected node sso_link_%s, got %q", connA, rec.Body.String())
			}
		})
	}
}

func TestUnlinkWithoutPrivilegedSession(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, mocks := newTestAPI(t, ctrl)
	session := passwordSession("s1")
	authenticatedAt := t0.Add(-2 * time.Hour)
	session.AuthenticatedAt = &authenticatedAt

	mocks.kratos.EXPECT().CheckSession(gomock.Any(), gomock.Any()).Times(1).Return(session, nil, nil)
	mocks.kratos.EXPECT().GetSettingsFlow(gomock.Any(), "sf", gomock.Any()).Times(1).Return(settingsFlow(), nil, nil)

	rec := httptest.NewRecorder()
	api.InterceptSettingsSubmission(rec, unlinkRequest(), "sf")

	expectErrorID(t, rec, kratos.SESSION_REFRESH_REQUIRED)
	expectedRedirect := "/ui/login?" + url.Values{"refresh": {"true"}, "return_to": {testRefreshReturnTo}}.Encode()
	expectRedirectTo(t, rec, expectedRedirect)
}

func TestUnlinkWhenSettingsFlowRedirects(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, mocks := newTestAPI(t, ctrl)
	redirectTo := "/ui/login?aal=aal2"

	mocks.kratos.EXPECT().CheckSession(gomock.Any(), gomock.Any()).Times(1).Return(privilegedSession(), nil, nil)
	mocks.kratos.EXPECT().GetSettingsFlow(gomock.Any(), "sf", gomock.Any()).Times(1).Return(nil, &kratos.BrowserLocationChangeRequired{RedirectTo: &redirectTo}, nil)

	rec := httptest.NewRecorder()
	api.InterceptSettingsSubmission(rec, unlinkRequest(), "sf")

	expectRedirectTo(t, rec, redirectTo)
}

func TestUnlinkFails(t *testing.T) {
	tests := []struct {
		name            string
		setupMocks      func(mocks *apiMocks)
		expectedStatus  int
		expectedErrorID string
	}{
		{
			name: "get settings flow",
			setupMocks: func(mocks *apiMocks) {
				mocks.kratos.EXPECT().GetSettingsFlow(gomock.Any(), "sf", gomock.Any()).Times(1).Return(nil, nil, errors.New("error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			// the frontend acts on the error of Kratos
			name: "get settings flow with an error of Kratos",
			setupMocks: func(mocks *apiMocks) {
				mocks.kratos.EXPECT().GetSettingsFlow(gomock.Any(), "sf", gomock.Any()).Times(1).Return(nil, nil, kratosError("self_service_flow_expired", http.StatusGone))
			},
			expectedStatus:  http.StatusGone,
			expectedErrorID: "self_service_flow_expired",
		},
		{
			name: "delete link",
			setupMocks: func(mocks *apiMocks) {
				mocks.kratos.EXPECT().GetSettingsFlow(gomock.Any(), "sf", gomock.Any()).Times(1).Return(settingsFlow(), nil, nil)
				mocks.sso.EXPECT().DeleteLink(gomock.Any(), connA, "iid").Times(1).Return(errors.New("error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			api, mocks := newTestAPI(t, ctrl)

			mocks.kratos.EXPECT().CheckSession(gomock.Any(), gomock.Any()).Times(1).Return(privilegedSession(), nil, nil)
			tt.setupMocks(mocks)

			rec := httptest.NewRecorder()
			api.InterceptSettingsSubmission(rec, unlinkRequest(), "sf")

			expectStatus(t, rec, tt.expectedStatus)
			if tt.expectedErrorID != "" {
				expectErrorID(t, rec, tt.expectedErrorID)
			}
		})
	}
}

func TestHydrateRefreshWithoutSession(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, _ := newTestAPI(t, ctrl)

	flow, ok := api.HydrateLoginFlow(httptest.NewRecorder(), getLoginFlowRequest(), refreshFlow())

	expectRendered(t, ok)
	expectNodes(t, flow, ssoNodeGroup, 0)
}

func TestHydrateRefreshFailOnLinks(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, mocks := newTestAPI(t, ctrl)

	mocks.kratos.EXPECT().CheckSession(gomock.Any(), gomock.Any()).Times(1).Return(companySession("s1"), nil, nil)
	mocks.sso.EXPECT().Links(gomock.Any(), "iid").Times(1).Return(nil, errors.New("error"))

	flow, ok := api.HydrateLoginFlow(httptest.NewRecorder(), withSessionCookie(getLoginFlowRequest()), refreshFlow())

	expectRendered(t, ok)
	if !hasMessage(flow, companyUnavailableID) {
		t.Fatalf("expected the message that company sign-in is unavailable")
	}
	expectNodes(t, flow, ssoNodeGroup, 0)
}

func TestRefreshPick(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, mocks := newTestAPI(t, ctrl)
	returnTo := BASE_URL + completePath + "?" + url.Values{"return_to": {testRefreshReturnTo}}.Encode()

	mocks.kratos.EXPECT().CheckSession(gomock.Any(), gomock.Any()).Times(1).Return(companySession("s-old"), nil, nil)
	mocks.sso.EXPECT().Links(gomock.Any(), "iid").Times(1).Return([]Link{{ConnectionID: connA, Label: "Acme", TenantID: testTenant}}, nil)
	// the company sign-in runs on a new flow that returns through completePath
	mocks.kratos.EXPECT().CreateBrowserLoginFlow(gomock.Any(), "", returnTo, "", false, gomock.Any()).Times(1).DoAndReturn(
		func(_ context.Context, _, _, _ string, _ bool, c []*http.Cookie) (*kClient.LoginFlow, []*http.Cookie, error) {
			if hasSessionCookie(c) {
				t.Fatalf("expected no session cookie")
			}
			return loginFlow("fresh", "", "", true), nil, nil
		},
	)
	mocks.expectStartAttempt(t, "fresh", AttemptRequest{TenantID: testTenant, Email: testEmail, ConnectionID: connA, Reauthenticate: true})

	rec := httptest.NewRecorder()
	_, answered := api.InterceptLoginSubmission(rec, withSessionCookie(refreshPickRequest(connA)), refreshFlow())

	expectAnswered(t, answered)
	expectStatus(t, rec, http.StatusOK)
	expectRedirectTo(t, rec, testCompanySignIn)

	signIn, err := mocks.store.GetSignIn(requestWith(findCookie(rec, signInCookieName)))
	expectNoError(t, err)
	// the account is not made a member of the tenant by signing in again
	expected := SignInCookie{Ticket: "ticket-1", PriorSessionID: "s-old", ReturnTo: testRefreshReturnTo, TenantID: testTenant, ConnectionID: connA}
	if signIn != expected {
		t.Fatalf("expected sign-in cookie %+v, got %+v", expected, signIn)
	}
}

func TestRefreshPickRefused(t *testing.T) {
	tests := []struct {
		name           string
		request        func() *http.Request
		flow           func() *kClient.LoginFlow
		setupMocks     func(mocks *apiMocks)
		expectedStatus int
	}{
		{
			name:    "connection not linked",
			request: func() *http.Request { return withSessionCookie(refreshPickRequest(connB)) },
			flow:    refreshFlow,
			setupMocks: func(mocks *apiMocks) {
				mocks.kratos.EXPECT().CheckSession(gomock.Any(), gomock.Any()).Times(1).Return(companySession("s1"), nil, nil)
				mocks.sso.EXPECT().Links(gomock.Any(), "iid").Times(1).Return([]Link{{ConnectionID: connA, Label: "Acme", TenantID: testTenant}}, nil)
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "another flow",
			request:        func() *http.Request { return withSessionCookie(refreshPickRequest(connA)) },
			flow:           func() *kClient.LoginFlow { return loginFlow("f", testLoginChallenge, testEmail, false) },
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "no session",
			request:        func() *http.Request { return refreshPickRequest(connA) },
			flow:           refreshFlow,
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			api, mocks := newTestAPI(t, ctrl)

			if tt.setupMocks != nil {
				tt.setupMocks(mocks)
			}

			rec := httptest.NewRecorder()
			_, answered := api.InterceptLoginSubmission(rec, tt.request(), tt.flow())

			expectAnswered(t, answered)
			expectStatus(t, rec, tt.expectedStatus)
		})
	}
}
