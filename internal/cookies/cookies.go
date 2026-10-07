// Copyright 2024 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

// Package cookies provides the shared FlowStateCookie type, the
// AuthCookieManagerInterface, and the production AuthCookieManager
// implementation. It is shared between the kratos and tenants packages.
package cookies

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"github.com/canonical/identity-platform-login-ui/internal/logging"
)

const (
	defaultCookiePath = "/"
	stateCookieName   = "login_ui_state"

	// NoTenantAvailable is a sentinel TenantID stored in FlowStateCookie
	// when multi-tenancy is enabled but the user has no tenants to choose
	// from. It distinguishes "selection completed with zero results" from
	// "selection not yet attempted" (empty string).
	NoTenantAvailable = "_none"
)

var epoch = time.Unix(0, 0).UTC()

// ChallengeHash returns the SHA-256 hex digest of loginChallenge.
// It is the canonical way to compute LoginChallengeHash values stored in
// FlowStateCookie, used by both the kratos and tenants packages.
func ChallengeHash(loginChallenge string) string {
	h := sha256.Sum256([]byte(loginChallenge))
	return hex.EncodeToString(h[:])
}

// FlowStateCookie holds per-flow UI state persisted across redirects in an
// encrypted browser cookie. Every field belongs to the login challenge of
// LoginChallengeHash: bind the cookie to another challenge only through
// RenewForChallenge.
type FlowStateCookie struct {
	LoginChallengeHash string `json:"lc,omitempty"`
	TotpSetup          bool   `json:"t,omitempty"`
	WebauthnSetup      bool   `json:"w,omitempty"`
	BackupCodeUsed     bool   `json:"bc,omitempty"`
	TenantID           string `json:"tid,omitempty"`
	// LoginStartedAt is when Kratos issued the first login flow submitted
	// for the challenge. The hash only says which login the cookie is for,
	// not that anyone signed in for it: it is set before any credential is
	// given.
	LoginStartedAt time.Time `json:"ls,omitzero"`
}

// AuthCookieManager is the production implementation of AuthCookieManagerInterface.
type AuthCookieManager struct {
	cookieTTL time.Duration
	encrypt   EncryptInterface
	logger    logging.LoggerInterface
}

// RenewForChallenge returns a new FlowStateCookie with LoginChallengeHash set
// for the given loginChallenge. TenantID and LoginStartedAt are carried
// forward only when the existing cookie was stored for the same challenge,
// preventing state pollution across different flows.
func (c FlowStateCookie) RenewForChallenge(loginChallenge string) FlowStateCookie {
	lcHash := ChallengeHash(loginChallenge)
	next := FlowStateCookie{LoginChallengeHash: lcHash}
	if c.LoginChallengeHash == lcHash {
		next.TenantID = c.TenantID
		next.LoginStartedAt = c.LoginStartedAt
	}
	return next
}

// StartLogin renews the cookie for loginChallenge and records when its login
// started, from the issue time of a Kratos login flow submitted for it. The
// first start recorded for a challenge is kept, so that a later flow of the
// same login cannot move it past a sign-in already made for it.
func (c FlowStateCookie) StartLogin(loginChallenge string, flowIssuedAt time.Time) FlowStateCookie {
	next := c.RenewForChallenge(loginChallenge)
	if next.LoginStartedAt.IsZero() {
		next.LoginStartedAt = flowIssuedAt
	}
	return next
}

// SignedInFor reports whether a session that authenticated at authenticatedAt
// counts as signed in for loginChallenge: it authenticated after the first
// login flow submitted for that challenge was issued. That does not prove the
// sign-in went through that flow: one made elsewhere in the same browser
// after the start counts too. Both times are stamped by Kratos, never by this
// service or the browser.
func (c FlowStateCookie) SignedInFor(loginChallenge string, authenticatedAt *time.Time) bool {
	return authenticatedAt != nil &&
		!c.LoginStartedAt.IsZero() &&
		c.LoginChallengeHash == ChallengeHash(loginChallenge) &&
		authenticatedAt.After(c.LoginStartedAt)
}

func (a *AuthCookieManager) SetStateCookie(w http.ResponseWriter, state FlowStateCookie) error {
	rawState, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return a.setCookie(w, stateCookieName, string(rawState), defaultCookiePath, a.cookieTTL, http.SameSiteLaxMode)
}

func (a *AuthCookieManager) GetStateCookie(r *http.Request) (FlowStateCookie, error) {
	var ret FlowStateCookie
	c, err := a.getCookie(r, stateCookieName)
	if c == "" || err != nil {
		return FlowStateCookie{}, err
	}
	err = json.Unmarshal([]byte(c), &ret)
	return ret, err
}

func (a *AuthCookieManager) ClearStateCookie(w http.ResponseWriter) {
	a.clearCookie(w, stateCookieName, defaultCookiePath)
}

func (a *AuthCookieManager) setCookie(w http.ResponseWriter, name, value string, path string, ttl time.Duration, sameSitePolicy http.SameSite) error {
	if value == "" {
		return nil
	}

	expires := time.Now().Add(ttl)

	encrypted, err := a.encrypt.Encrypt(value)
	if err != nil {
		a.logger.Errorf("cannot encrypt cookie value: %v", err)
		return err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    encrypted,
		Path:     path,
		Domain:   "",
		Expires:  expires,
		MaxAge:   int(ttl.Seconds()),
		HttpOnly: true,
		Secure:   true,
		SameSite: sameSitePolicy,
	})
	return nil
}

func (a *AuthCookieManager) clearCookie(w http.ResponseWriter, name string, path string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     path,
		Domain:   "",
		Expires:  epoch,
		MaxAge:   -1,
		Secure:   true,
		HttpOnly: true,
	})
}

func (a *AuthCookieManager) getCookie(r *http.Request, name string) (string, error) {
	cookie, err := r.Cookie(name)
	if err != nil {
		// This means that the cookie does not exist, not a real error
		return "", nil
	}

	value, err := a.encrypt.Decrypt(cookie.Value)
	if err != nil {
		a.logger.Errorf("cannot decrypt cookie value: %v", err)
		return "", err
	}
	return value, nil
}

// NewAuthCookieManager constructs an AuthCookieManager with the given TTL,
// encryption backend, and logger.
func NewAuthCookieManager(
	cookieTTLSeconds int,
	encrypt EncryptInterface,
	logger logging.LoggerInterface,
) *AuthCookieManager {
	a := new(AuthCookieManager)
	a.cookieTTL = time.Duration(cookieTTLSeconds) * time.Second
	a.encrypt = encrypt
	a.logger = logger
	return a
}
