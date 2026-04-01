// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0

package cookies

import "net/http"

// EncryptInterface abstracts string encryption for the cookie manager.
type EncryptInterface interface {
	// Encrypt a plain text string, returns the encrypted string in hex format or an error
	Encrypt(string) (string, error)
	// Decrypt a hex string, returns the decrypted string or an error
	Decrypt(string) (string, error)
}

// AuthCookieManagerInterface describes operations on the encrypted state cookie.
type AuthCookieManagerInterface interface {
	// SetStateCookie sets the nonce cookie on the response with the specified duration as MaxAge
	SetStateCookie(http.ResponseWriter, FlowStateCookie) error
	// GetStateCookie returns the string value of the nonce cookie if present, or empty string otherwise
	GetStateCookie(*http.Request) (FlowStateCookie, error)
	// ClearStateCookie sets the expiration of the cookie to epoch
	ClearStateCookie(http.ResponseWriter)
}
