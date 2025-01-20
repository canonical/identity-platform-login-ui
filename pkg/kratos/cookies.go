// Copyright 2024 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0

package kratos

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/canonical/identity-platform-login-ui/internal/logging"
)

const (
	defaultCookiePath = "/"
	stateCookieName   = "login_ui_state"
)

var (
	epoch = time.Unix(0, 0).UTC()
)

type AuthCookieManager struct {
	cookieTTL time.Duration
	encrypt   EncryptInterface

	logger logging.LoggerInterface
}

type FlowStateCookie struct {
	LoginChallengeHash  string `json:"lc,omitempty"`
	KratosSessionIdHash string `json:"sh",omitempty`
	TotpSetup           bool   `json:"t,omitempty"`
	BackupCodeUsed      bool   `json:"bc,omitempty"`
	OidcLogin           bool   `json:"oi,omitempty"`
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
		a.logger.Errorf("can't encrypt cookie value, %v", err)
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
		a.logger.Errorf("can't decrypt cookie value, %v", err)
		return "", err
	}
	return value, nil
}

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
