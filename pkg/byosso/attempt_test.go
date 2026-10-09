// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package byosso

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/canonical/identity-platform-login-ui/internal/cookies"
	"github.com/canonical/identity-platform-login-ui/pkg/kratos"
)

func TestStartAttemptWithLoginChallenge(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, mocks := newTestAPI(t, ctrl)
	flow := loginFlow("f", testLoginChallenge, testEmail, false)
	flowCookies := []*http.Cookie{{Name: "csrf_token", Value: "csrf"}}

	mocks.expectStartAttempt(t, "f", AttemptRequest{TenantID: testTenant, Email: testEmail, ConnectionID: connA, Reauthenticate: true})
	mocks.state.EXPECT().GetStateCookie(gomock.Any()).Times(1).Return(cookies.FlowStateCookie{}, nil)
	mocks.state.EXPECT().SetStateCookie(gomock.Any(), chosenStateFor(testTenant)).Times(1).Return(nil)

	rec := httptest.NewRecorder()
	redirectTo, ok := api.startAttempt(rec, withSessionCookie(updateLoginFlowRequest(`{}`)), flow, flowCookies, attempt{
		loginChallenge: testLoginChallenge,
		tenantID:       testTenant,
		email:          testEmail,
		connectionID:   connA,
		reauthenticate: true,
		tenantChosen:   true,
		prior:          "s1",
	})

	if !ok {
		t.Fatalf("expected the attempt to start")
	}
	if redirectTo != testCompanySignIn {
		t.Fatalf("expected redirect to %s, got %s", testCompanySignIn, redirectTo)
	}

	signIn, err := mocks.store.GetSignIn(requestWith(findCookie(rec, signInCookieName)))
	expectNoError(t, err)
	expectedSignIn := SignInCookie{
		Ticket:             "ticket-1",
		PriorSessionID:     "s1",
		LoginChallengeHash: cookies.ChallengeHash(testLoginChallenge),
		TenantID:           testTenant,
		ConnectionID:       connA,
	}
	if signIn != expectedSignIn {
		t.Fatalf("expected sign-in cookie %+v, got %+v", expectedSignIn, signIn)
	}

	fresh, err := mocks.store.GetFresh(requestWith(findCookie(rec, freshCookieName)))
	expectNoError(t, err)
	expectedFresh := FreshCookie{LoginChallengeHash: cookies.ChallengeHash(testLoginChallenge), PriorSessionID: "s1", StartedAt: t0}
	if fresh != expectedFresh {
		t.Fatalf("expected fresh cookie %+v, got %+v", expectedFresh, fresh)
	}
	if c := findCookie(rec, "csrf_token"); c == nil {
		t.Fatalf("expected the cookies of the flow to be set")
	}
	if sessionCookie := findCookie(rec, kratos.KRATOS_SESSION_COOKIE_NAME); sessionCookie == nil || sessionCookie.Value != "" {
		t.Fatalf("expected the session cookie to be unset, got %v", sessionCookie)
	}
}

func TestStartAttemptWithoutLoginChallenge(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, mocks := newTestAPI(t, ctrl)
	flow := loginFlow("f", "", "", true)

	// no tenant is bound and no first factor is marked without a login challenge
	mocks.expectStartAttempt(t, "f", AttemptRequest{TenantID: testTenant, Email: testEmail, ConnectionID: connA})

	rec := httptest.NewRecorder()
	_, ok := api.startAttempt(rec, updateLoginFlowRequest(`{}`), flow, nil, attempt{
		tenantID:     testTenant,
		email:        testEmail,
		connectionID: connA,
		join:         true,
		returnTo:     "https://portal.example/welcome",
		linkTicket:   "ticket-0",
	})

	if !ok {
		t.Fatalf("expected the attempt to start")
	}

	signIn, err := mocks.store.GetSignIn(requestWith(findCookie(rec, signInCookieName)))
	expectNoError(t, err)
	expectedSignIn := SignInCookie{
		Ticket:       "ticket-1",
		ReturnTo:     "https://portal.example/welcome",
		TenantID:     testTenant,
		ConnectionID: connA,
		LinkTicket:   "ticket-0",
		Join:         true,
	}
	if signIn != expectedSignIn {
		t.Fatalf("expected sign-in cookie %+v, got %+v", expectedSignIn, signIn)
	}
	if c := findCookie(rec, freshCookieName); c != nil {
		t.Fatalf("expected no fresh cookie, got %v", c)
	}
}

func TestStartAttemptFailOnStartAttempt(t *testing.T) {
	tests := []struct {
		name            string
		err             error
		expectedStatus  int
		expectedErrorID string
	}{
		{name: "not applicable", err: fmt.Errorf("cannot start a company sign-in: %w", errNotApplicable), expectedStatus: http.StatusForbidden, expectedErrorID: ssoNotApplicableError},
		{name: "sso-service fails", err: errors.New("error"), expectedStatus: http.StatusServiceUnavailable, expectedErrorID: ssoUnavailableError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			api, mocks := newTestAPI(t, ctrl)

			mocks.sso.EXPECT().StartAttempt(gomock.Any(), gomock.Any()).Times(1).Return("", tt.err)

			rec := httptest.NewRecorder()
			_, ok := api.startAttempt(rec, updateLoginFlowRequest(`{}`), loginFlow("f", testLoginChallenge, testEmail, false), nil, attempt{
				loginChallenge: testLoginChallenge,
				tenantID:       testTenant,
				email:          testEmail,
				connectionID:   connA,
			})

			if ok {
				t.Fatalf("expected the attempt not to start")
			}
			expectStatus(t, rec, tt.expectedStatus)
			expectErrorID(t, rec, tt.expectedErrorID)
		})
	}
}

func TestStartAttemptFailOnUpdateLoginFlow(t *testing.T) {
	tests := []struct {
		name            string
		err             error
		expectedStatus  int
		expectedErrorID string
	}{
		{name: "error", err: errors.New("error"), expectedStatus: http.StatusInternalServerError},
		// the frontend acts on the error of Kratos
		{name: "error of Kratos", err: kratosError("self_service_flow_expired", http.StatusGone), expectedStatus: http.StatusBadRequest, expectedErrorID: "self_service_flow_expired"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			api, mocks := newTestAPI(t, ctrl)

			mocks.sso.EXPECT().StartAttempt(gomock.Any(), gomock.Any()).Times(1).Return("ticket-1", nil)
			mocks.kratos.EXPECT().UpdateLoginFlow(gomock.Any(), "f", gomock.Any(), gomock.Any()).Times(1).Return(nil, nil, nil, tt.err)

			rec := httptest.NewRecorder()
			_, ok := api.startAttempt(rec, updateLoginFlowRequest(`{}`), loginFlow("f", testLoginChallenge, testEmail, false), nil, attempt{
				loginChallenge: testLoginChallenge,
				tenantID:       testTenant,
				email:          testEmail,
				connectionID:   connA,
			})

			if ok {
				t.Fatalf("expected the attempt not to start")
			}
			expectStatus(t, rec, tt.expectedStatus)
			if tt.expectedErrorID != "" {
				expectErrorID(t, rec, tt.expectedErrorID)
			}
			if c := findCookie(rec, signInCookieName); c != nil {
				t.Fatalf("expected no sign-in cookie, got %v", c)
			}
		})
	}
}

func TestCompletePendingNotApplicable(t *testing.T) {
	tests := []struct {
		name    string
		receipt string
	}{
		// the attempt did not end at this account
		{name: "receipt sso-service refuses", receipt: "receipt"},
		// sso-service is asked all the same, with an empty receipt
		{name: "no receipt cookie"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			api, mocks := newTestAPI(t, ctrl)
			session := companySession("s-new")
			r := signInRequest(mocks, SignInCookie{Ticket: "ticket"})
			if tt.receipt != "" {
				withReceipt(r, "ticket", tt.receipt)
			}

			mocks.sso.EXPECT().CompleteAttempt(gomock.Any(), "ticket", "iid", tt.receipt).Times(1).Return(nil, fmt.Errorf("cannot complete the company sign-in: %w", errNotApplicable))

			rec := httptest.NewRecorder()
			provenance, completed, err := api.completePending(rec, r, session)

			expectNoError(t, err)
			if provenance != "" {
				t.Fatalf("expected no provenance, got %s", provenance)
			}
			if completed != nil {
				t.Fatalf("expected no completed attempt, got %+v", completed)
			}
			if signInCookie := findCookie(rec, signInCookieName); signInCookie == nil || signInCookie.MaxAge >= 0 {
				t.Fatalf("expected the sign-in cookie to be cleared, got %v", signInCookie)
			}
			expectReceiptCleared(t, rec, "ticket")
			if c := findCookie(rec, provenanceCookieName); c != nil {
				t.Fatalf("expected no provenance cookie, got %v", c)
			}
		})
	}
}

func TestCompletePendingWithAttemptBeingLinked(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, mocks := newTestAPI(t, ctrl)
	session := companySession("s-new")
	r := signInRequest(mocks, SignInCookie{Ticket: "ticket-a", LinkTicket: "ticket-b"})
	withReceipt(r, "ticket-a", "receipt-a")
	withReceipt(r, "ticket-b", "receipt-b")

	// each ticket goes with its own receipt: the one being linked, which is
	// refused here, then the one that proved the account
	gomock.InOrder(
		mocks.sso.EXPECT().CompleteAttempt(gomock.Any(), "ticket-b", "iid", "receipt-b").Times(1).Return(nil, fmt.Errorf("cannot complete the company sign-in: %w", errNotApplicable)),
		mocks.sso.EXPECT().CompleteAttempt(gomock.Any(), "ticket-a", "iid", "receipt-a").Times(1).Return(&Completion{ConnectionID: connA, TenantID: testTenant}, nil),
	)

	rec := httptest.NewRecorder()
	provenance, _, err := api.completePending(rec, r, session)

	expectNoError(t, err)
	if provenance != connA {
		t.Fatalf("expected provenance of connection %s, got %s", connA, provenance)
	}
	expectReceiptCleared(t, rec, "ticket-a")
	expectReceiptCleared(t, rec, "ticket-b")
}

func TestCompletePendingWithoutCompanySignIn(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, mocks := newTestAPI(t, ctrl)
	r := signInRequest(mocks, SignInCookie{Ticket: "ticket"})

	rec := httptest.NewRecorder()
	provenance, completed, err := api.completePending(rec, r, passwordSession("s-new"))

	expectNoError(t, err)
	if provenance != "" {
		t.Fatalf("expected no provenance, got %s", provenance)
	}
	if completed != nil {
		t.Fatalf("expected no completed attempt, got %+v", completed)
	}
	if signInCookie := findCookie(rec, signInCookieName); signInCookie == nil || signInCookie.MaxAge >= 0 {
		t.Fatalf("expected the sign-in cookie to be cleared, got %v", signInCookie)
	}
}

func TestIsFresh(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, mocks := newTestAPI(t, ctrl)
	// the first factor of these sessions completes as the flow is issued
	flow := loginFlow("f", testLoginChallenge, testEmail, false)
	r := withSessionCookie(updateLoginFlowRequest(`{"method":"password","identifier":"bob@acme.example","password":"test"}`))

	mocks.kratos.EXPECT().CheckSession(gomock.Any(), gomock.Any()).Times(1).Return(passwordSession("s1"), nil, nil)

	rec := httptest.NewRecorder()
	next, _ := api.InterceptLoginSubmission(rec, r, flow)

	// a password completes the first factor in the request that marked it
	if !api.isFresh(next, testLoginChallenge, passwordSession("s2")) {
		t.Fatalf("expected the session the first factor produced to be fresh")
	}
	if api.isFresh(next, testLoginChallenge, passwordSession("s1")) {
		t.Fatalf("expected the session the browser had not to be fresh")
	}
	if api.isFresh(next, "another_challenge", passwordSession("s2")) {
		t.Fatalf("expected the session not to be fresh for another login challenge")
	}

	// a later request carries the mark in the fresh cookie
	later := createLoginFlowRequest()
	later.AddCookie(findCookie(rec, freshCookieName))
	if !api.isFresh(later, testLoginChallenge, passwordSession("s2")) {
		t.Fatalf("expected the session to be fresh with the fresh cookie")
	}
	if api.isFresh(createLoginFlowRequest(), testLoginChallenge, passwordSession("s2")) {
		t.Fatalf("expected the session not to be fresh without a mark")
	}
}
