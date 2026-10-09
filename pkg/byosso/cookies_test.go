// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package byosso

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/mock/gomock"
)

// newTestCookieStore returns the cookie store of an API built on mocks, with
// its clock at t0.
func newTestCookieStore(t *testing.T, ctrl *gomock.Controller) *CookieStore {
	t.Helper()
	_, mocks := newTestAPI(t, ctrl)
	mocks.store.now = func() time.Time { return t0 }
	return mocks.store
}

func TestSetSignIn(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	store := newTestCookieStore(t, ctrl)
	expected := SignInCookie{Ticket: "ticket", PriorSessionID: "s-old", LoginChallengeHash: "hash", ReturnTo: "https://portal.example/welcome", TenantID: testTenant, ConnectionID: connA, LinkTicket: "ticket-b", Join: true}

	rec := httptest.NewRecorder()
	err := store.SetSignIn(rec, expected)

	expectNoError(t, err)

	c := findCookie(rec, "login_ui_sso_signin")
	if c == nil {
		t.Fatalf("expected cookie login_ui_sso_signin to be set")
	}
	if !c.HttpOnly {
		t.Fatalf("expected the cookie to be HttpOnly")
	}
	if !c.Secure {
		t.Fatalf("expected the cookie to be Secure")
	}
	if c.SameSite != http.SameSiteLaxMode {
		t.Fatalf("expected the cookie to be SameSite=Lax, got %v", c.SameSite)
	}
	if c.Path != "/" {
		t.Fatalf("expected path /, got %s", c.Path)
	}
	// the cookie lives as long as the attempt
	if !c.Expires.Equal(t0.Add(30 * time.Minute)) {
		t.Fatalf("expected the cookie to expire at %v, got %v", t0.Add(30*time.Minute), c.Expires)
	}

	signIn, err := store.GetSignIn(requestWith(c))
	expectNoError(t, err)
	if signIn != expected {
		t.Fatalf("expected sign-in cookie %+v, got %+v", expected, signIn)
	}
}

func TestSetProvenance(t *testing.T) {
	expiresAt := t0.Add(8 * time.Hour)
	expired := t0.Add(-time.Hour)

	tests := []struct {
		name            string
		expiresAt       *time.Time
		expectedExpires time.Time
	}{
		{name: "session with an expiry", expiresAt: &expiresAt, expectedExpires: expiresAt},
		{name: "session without expiry", expiresAt: nil, expectedExpires: t0.Add(time.Hour)},
		{name: "expired session", expiresAt: &expired, expectedExpires: t0.Add(time.Hour)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			store := newTestCookieStore(t, ctrl)
			session := companySession("s1")
			session.ExpiresAt = tt.expiresAt
			expected := ProvenanceCookie{SessionID: "s1", ConnectionID: connA}

			rec := httptest.NewRecorder()
			err := store.SetProvenance(rec, expected, session)

			expectNoError(t, err)

			c := findCookie(rec, "login_ui_sso_provenance")
			if c == nil {
				t.Fatalf("expected cookie login_ui_sso_provenance to be set")
			}
			// the cookie lives until the session expires
			if !c.Expires.Equal(tt.expectedExpires) {
				t.Fatalf("expected the cookie to expire at %v, got %v", tt.expectedExpires, c.Expires)
			}

			provenance, err := store.GetProvenance(requestWith(c))
			expectNoError(t, err)
			if provenance != expected {
				t.Fatalf("expected provenance cookie %+v, got %+v", expected, provenance)
			}
		})
	}
}

func TestGetFreshWithValueOfAnotherCookie(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	store := newTestCookieStore(t, ctrl)

	// the sign-in cookie holds the fields of a fresh mark under the same names
	rec := httptest.NewRecorder()
	expectNoError(t, store.SetSignIn(rec, SignInCookie{Ticket: "ticket", LoginChallengeHash: "hash"}))
	signIn := findCookie(rec, signInCookieName)

	mark, err := store.GetFresh(requestWith(&http.Cookie{Name: freshCookieName, Value: signIn.Value}))

	if err == nil {
		t.Fatalf("expected error not nil")
	}
	if mark != (FreshCookie{}) {
		t.Fatalf("expected an empty fresh cookie, got %+v", mark)
	}
}

func TestClearReceipt(t *testing.T) {
	rec := httptest.NewRecorder()
	clearReceipt(rec, "ticket")

	// the name is the one sso-service sets: the prefix and the first 16 hex
	// digits of the SHA-256 of the ticket. A browser takes the change of a
	// __Host- cookie only with Path=/ and Secure, and no Domain.
	expected := "__Host-sso_receipt_14069429150abcbf=; Path=/; Expires=Thu, 01 Jan 1970 00:00:00 GMT; Max-Age=0; HttpOnly; Secure; SameSite=Lax"
	if header := rec.Header().Get("Set-Cookie"); header != expected {
		t.Fatalf("expected Set-Cookie %q, got %q", expected, header)
	}
}

func TestGetProvenanceFailOnDecrypt(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	store := newTestCookieStore(t, ctrl)
	r := requestWith(&http.Cookie{Name: "login_ui_sso_provenance", Value: "plain"})

	provenance, err := store.GetProvenance(r)

	if err == nil {
		t.Fatalf("expected error not nil")
	}
	if provenance != (ProvenanceCookie{}) {
		t.Fatalf("expected an empty provenance cookie, got %+v", provenance)
	}
}
