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

	"github.com/canonical/identity-platform-login-ui/internal/cookies"
)

// linkingFlow builds the account-linking login flow of Kratos for testEmail,
// with the nodes of the sign-ins the account has.
func linkingFlow(nodes ...kClient.UiNode) *kClient.LoginFlow {
	flow := kClient.NewLoginFlowWithDefaults()
	flow.Id = "linking"
	loginChallenge := testLoginChallenge
	flow.Oauth2LoginChallenge = &loginChallenge
	flow.Ui.Nodes = append([]kClient.UiNode{inputNode("default", "csrf_token", "csrf-1"), inputNode("default", "identifier", testEmail)}, nodes...)
	flow.Ui.Messages = []kClient.UiText{*kClient.NewUiText(signInAndLink, "You tried to sign in with ...", "info")}
	return flow
}

func linkingPickRequest(connectionID string) *http.Request {
	return jsonRequest(http.MethodPost, "/api/kratos/self-service/login?flow=linking", `{"sso_account_link":"`+connectionID+`"}`)
}

func TestHydrateAccountLinking(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, mocks := newTestAPI(t, ctrl)
	r := withCookies(jsonRequest(http.MethodGet, "/api/kratos/self-service/login/flows?id=linking", ""), func(w http.ResponseWriter) {
		_ = mocks.store.SetSignIn(w, SignInCookie{Ticket: "ticket-b", TenantID: testTenant, ConnectionID: connB})
	})

	mocks.identities.EXPECT().IdentityID(gomock.Any(), testEmail).Times(1).Return("iid", nil)
	mocks.sso.EXPECT().Links(gomock.Any(), "iid").Times(1).Return([]Link{{ConnectionID: connA, Label: "Acme"}, {ConnectionID: connB, Label: "Beta"}}, nil)

	flow, ok := api.HydrateLoginFlow(httptest.NewRecorder(), r, linkingFlow(inputNode("password", "password", "")))

	expectRendered(t, ok)

	// the other company sign-in of the account, never the one being linked
	links := nodesOf(flow, ssoNodeGroup)
	if len(links) != 1 {
		t.Fatalf("expected 1 company sign-in node, got %d", len(links))
	}
	attrs := links[0].Attributes.UiNodeInputAttributes
	if attrs.Name != "sso_account_link" {
		t.Fatalf("expected input sso_account_link, got %s", attrs.Name)
	}
	if attrs.Value != connA {
		t.Fatalf("expected value %s, got %v", connA, attrs.Value)
	}
	expectNodes(t, flow, "password", 1)
	if hasMessage(flow, recoverPromptID) {
		t.Fatalf("expected no recovery prompt")
	}
}

func TestHydrateAccountLinkingFailOnLinks(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, mocks := newTestAPI(t, ctrl)

	mocks.identities.EXPECT().IdentityID(gomock.Any(), testEmail).Times(1).Return("iid", nil)
	mocks.sso.EXPECT().Links(gomock.Any(), "iid").Times(1).Return(nil, errors.New("error"))

	flow, ok := api.HydrateLoginFlow(httptest.NewRecorder(), getLoginFlowRequest(), linkingFlow(inputNode("password", "password", "")))

	expectRendered(t, ok)
	if !hasMessage(flow, companyUnavailableID) {
		t.Fatalf("expected the message that company sign-in is unavailable")
	}
}

func TestLinkingPick(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, mocks := newTestAPI(t, ctrl)
	r := withCookies(linkingPickRequest(connA), func(w http.ResponseWriter) {
		_ = mocks.store.SetSignIn(w, SignInCookie{
			Ticket:             "ticket-b",
			TenantID:           testTenant,
			ConnectionID:       connB,
			LoginChallengeHash: cookies.ChallengeHash(testLoginChallenge),
			ReturnTo:           "https://portal.example/welcome",
			Join:               true,
		})
	})

	mocks.identities.EXPECT().IdentityID(gomock.Any(), testEmail).Times(1).Return("iid", nil)
	mocks.sso.EXPECT().Links(gomock.Any(), "iid").Times(1).Return([]Link{{ConnectionID: connA, Label: "Acme"}}, nil)
	mocks.expectStartAttempt(t, "linking", AttemptRequest{TenantID: testTenant, Email: testEmail, ConnectionID: connA})
	mocks.state.EXPECT().GetStateCookie(gomock.Any()).Times(1).Return(stateFor(testTenant), nil)
	mocks.state.EXPECT().SetStateCookie(gomock.Any(), stateFor(testTenant)).Times(1).Return(nil)

	rec := httptest.NewRecorder()
	_, answered := api.InterceptLoginSubmission(rec, r, linkingFlow())

	expectAnswered(t, answered)
	expectStatus(t, rec, http.StatusOK)
	expectRedirectTo(t, rec, testCompanySignIn)
	if label := responseBody(t, rec)["redirect_label"]; label != "Acme" {
		t.Fatalf("expected redirect label Acme, got %v", label)
	}

	// the sign-in cookie keeps the attempt being linked
	signIn, err := mocks.store.GetSignIn(requestWith(findCookie(rec, signInCookieName)))
	expectNoError(t, err)
	expected := SignInCookie{
		Ticket:             "ticket-1",
		LinkTicket:         "ticket-b",
		TenantID:           testTenant,
		ConnectionID:       connA,
		LoginChallengeHash: cookies.ChallengeHash(testLoginChallenge),
		ReturnTo:           "https://portal.example/welcome",
		Join:               true,
	}
	if signIn != expected {
		t.Fatalf("expected sign-in cookie %+v, got %+v", expected, signIn)
	}
}

func TestLinkingPickRefused(t *testing.T) {
	// the account is linked to connA
	linked := func(mocks *apiMocks) {
		mocks.identities.EXPECT().IdentityID(gomock.Any(), testEmail).Times(1).Return("iid", nil)
		mocks.sso.EXPECT().Links(gomock.Any(), "iid").Times(1).Return([]Link{{ConnectionID: connA, Label: "Acme"}}, nil)
	}

	tests := []struct {
		name           string
		flow           func() *kClient.LoginFlow
		connectionID   string
		signIn         *SignInCookie
		setupMocks     func(mocks *apiMocks)
		expectedStatus int
	}{
		{
			name:           "connection not linked",
			flow:           func() *kClient.LoginFlow { return linkingFlow() },
			connectionID:   connB,
			setupMocks:     linked,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "another flow",
			flow:           func() *kClient.LoginFlow { return loginFlow("f", testLoginChallenge, testEmail, false) },
			connectionID:   connA,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "no company sign-in started",
			flow:           func() *kClient.LoginFlow { return linkingFlow() },
			connectionID:   connA,
			signIn:         &SignInCookie{},
			setupMocks:     linked,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "the connection being linked",
			flow:           func() *kClient.LoginFlow { return linkingFlow() },
			connectionID:   connA,
			signIn:         &SignInCookie{Ticket: "ticket-a", TenantID: testTenant, ConnectionID: connA},
			setupMocks:     linked,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			api, mocks := newTestAPI(t, ctrl)
			r := linkingPickRequest(tt.connectionID)
			if tt.signIn != nil {
				r = withCookies(r, func(w http.ResponseWriter) {
					_ = mocks.store.SetSignIn(w, *tt.signIn)
				})
			}

			if tt.setupMocks != nil {
				tt.setupMocks(mocks)
			}

			rec := httptest.NewRecorder()
			_, answered := api.InterceptLoginSubmission(rec, r, tt.flow())

			expectAnswered(t, answered)
			expectStatus(t, rec, tt.expectedStatus)
		})
	}
}

func TestLinkingPickFailOnLinks(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, mocks := newTestAPI(t, ctrl)

	mocks.identities.EXPECT().IdentityID(gomock.Any(), testEmail).Times(1).Return("", errors.New("error"))

	rec := httptest.NewRecorder()
	_, answered := api.InterceptLoginSubmission(rec, linkingPickRequest(connA), linkingFlow())

	expectAnswered(t, answered)
	expectStatus(t, rec, http.StatusServiceUnavailable)
	expectErrorID(t, rec, ssoUnavailableError)
}
