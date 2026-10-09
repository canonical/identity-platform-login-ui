// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package byosso

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	kClient "github.com/ory/kratos-client-go/v25"

	"github.com/canonical/identity-platform-login-ui/internal/cookies"
)

// SignInCookie is the cookie of a company sign-in in progress.
type SignInCookie struct {
	Ticket string `json:"t,omitempty"`
	// PriorSessionID is the ID of the Kratos session the browser had when
	// the company sign-in started, "" when it had none.
	PriorSessionID string `json:"ps,omitempty"`
	// LoginChallengeHash binds the attempt to a Hydra login request, if any.
	LoginChallengeHash string `json:"lc,omitempty"`
	// ReturnTo is where completePath sends the browser. It came from a
	// Kratos flow, so Kratos allowed it.
	ReturnTo     string `json:"rt,omitempty"`
	TenantID     string `json:"tid,omitempty"`
	ConnectionID string `json:"cid,omitempty"`
	// LinkTicket is the attempt Kratos is adding to the account, when Ticket
	// is a company sign-in the account already had, started from the
	// account-linking page to prove the account. The session that comes back
	// includes both, and the connection of LinkTicket is the one signed in
	// for.
	LinkTicket string `json:"lt,omitempty"`
	// Join marks the company sign-in of a registration. With no Hydra login
	// request, completePath makes the account a member of the tenant.
	Join bool `json:"j,omitempty"`
}

// ProvenanceCookie records the connection behind the first factor of a
// session. It counts only while SessionID is the session of the browser.
type ProvenanceCookie struct {
	SessionID    string `json:"sid,omitempty"`
	ConnectionID string `json:"c,omitempty"`
}

// FreshCookie marks the first factor started for a login challenge.
type FreshCookie struct {
	LoginChallengeHash string `json:"lc,omitempty"`
	PriorSessionID     string `json:"ps,omitempty"`
	// StartedAt is when Kratos issued the login flow the first factor began on.
	StartedAt time.Time `json:"at"`
}

// RegistrationCookie carries the tenant a registration is sent to, to the
// login flow that offers its company sign-ins.
type RegistrationCookie struct {
	FlowID   string `json:"f,omitempty"`
	TenantID string `json:"tid,omitempty"`
	Email    string `json:"e,omitempty"`
	ReturnTo string `json:"rt,omitempty"`
}

// CookieStore holds the cookies of company sign-ins. They are HttpOnly,
// Secure and SameSite=Lax, as they must arrive on the redirect chain that
// starts at the IdP.
type CookieStore struct {
	encrypt cookies.EncryptInterface
	now     func() time.Time
}

func NewCookieStore(encrypt cookies.EncryptInterface) *CookieStore {
	return &CookieStore{encrypt: encrypt, now: time.Now}
}

func (s *CookieStore) SetSignIn(w http.ResponseWriter, c SignInCookie) error {
	return s.set(w, signInCookieName, c, s.now().Add(attemptTTL))
}

func (s *CookieStore) GetSignIn(r *http.Request) (SignInCookie, error) {
	var c SignInCookie
	return c, s.get(r, signInCookieName, &c)
}

func (s *CookieStore) ClearSignIn(w http.ResponseWriter) { clearCookie(w, signInCookieName) }

func (s *CookieStore) SetProvenance(w http.ResponseWriter, c ProvenanceCookie, session *kClient.Session) error {
	return s.set(w, provenanceCookieName, c, s.sessionExpiry(session))
}

func (s *CookieStore) GetProvenance(r *http.Request) (ProvenanceCookie, error) {
	var c ProvenanceCookie
	return c, s.get(r, provenanceCookieName, &c)
}

func (s *CookieStore) SetFresh(w http.ResponseWriter, c FreshCookie) error {
	return s.set(w, freshCookieName, c, s.now().Add(attemptTTL))
}

func (s *CookieStore) GetFresh(r *http.Request) (FreshCookie, error) {
	var c FreshCookie
	return c, s.get(r, freshCookieName, &c)
}

func (s *CookieStore) ClearFresh(w http.ResponseWriter) { clearCookie(w, freshCookieName) }

func (s *CookieStore) SetRegistration(w http.ResponseWriter, c RegistrationCookie) error {
	return s.set(w, registrationCookieName, c, s.now().Add(attemptTTL))
}

func (s *CookieStore) GetRegistration(r *http.Request) (RegistrationCookie, error) {
	var c RegistrationCookie
	return c, s.get(r, registrationCookieName, &c)
}

func (s *CookieStore) ClearRegistration(w http.ResponseWriter) {
	clearCookie(w, registrationCookieName)
}

// sessionExpiry returns when the Kratos session expires.
func (s *CookieStore) sessionExpiry(session *kClient.Session) time.Time {
	if session != nil && session.ExpiresAt != nil && session.ExpiresAt.After(s.now()) {
		return *session.ExpiresAt
	}
	return s.now().Add(defaultSessionTTL)
}

// sealedAs is what a cookie's sealed value starts with: its name, so that the
// value of one cookie does not open as another.
func sealedAs(name string) string {
	return name + "\x00"
}

func (s *CookieStore) set(w http.ResponseWriter, name string, v any, expires time.Time) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	enc, err := s.encrypt.Encrypt(sealedAs(name) + string(raw))
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    enc,
		Path:     "/",
		Expires:  expires,
		MaxAge:   int(time.Until(expires).Seconds()),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
	return nil
}

// get leaves v zero when the cookie is absent. A cookie that does not
// decrypt, or was sealed under another name, is an error, and v stays zero.
func (s *CookieStore) get(r *http.Request, name string, v any) error {
	c, err := r.Cookie(name)
	if err != nil || c.Value == "" {
		return nil
	}
	plain, err := s.encrypt.Decrypt(c.Value)
	if err != nil {
		return err
	}
	raw, ok := strings.CutPrefix(plain, sealedAs(name))
	if !ok {
		return fmt.Errorf("cookie %s holds a value sealed for another cookie", name)
	}
	return json.Unmarshal([]byte(raw), v)
}

func receiptCookieName(ticket string) string {
	digest := sha256.Sum256([]byte(ticket))
	return receiptCookiePrefix + hex.EncodeToString(digest[:8])
}

// receiptFor returns the receipt the browser holds for the ticket, "" when
// it holds none.
func receiptFor(r *http.Request, ticket string) string {
	c, err := r.Cookie(receiptCookieName(ticket))
	if err != nil {
		return ""
	}
	return c.Value
}

// clearReceipt clears the receipt of the ticket. A browser takes the change
// of a __Host- cookie only with Path=/ and Secure, which clearCookie sets.
func clearReceipt(w http.ResponseWriter, ticket string) {
	clearCookie(w, receiptCookieName(ticket))
}

func clearCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0).UTC(),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}
