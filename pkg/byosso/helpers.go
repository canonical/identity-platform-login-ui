// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package byosso

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"maps"
	"net/http"
	"net/url"
	"strings"
	"unicode"

	kClient "github.com/ory/kratos-client-go/v25"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	httpHelpers "github.com/canonical/identity-platform-login-ui/internal/misc/http"
	"github.com/canonical/identity-platform-login-ui/pkg/kratos"
)

// redirect tells the browser where to go next. RedirectLabel names the
// company sign-in it is sent to.
type redirect struct {
	kratos.BrowserLocationChangeRequired
	RedirectLabel string                 `json:"redirect_label,omitempty"`
	ContinueWith  []kClient.ContinueWith `json:"continue_with,omitempty"`
}

// GetCode answers every redirect with a 200, whatever the error it carries:
// the frontend follows these redirects from the body.
func (r *redirect) GetCode() int {
	return http.StatusOK
}

func newRedirect(to string) *redirect {
	return &redirect{BrowserLocationChangeRequired: kratos.BrowserLocationChangeRequired{RedirectTo: &to}}
}

func (r *redirect) withError(id string) *redirect {
	r.Error = &kClient.GenericError{Id: &id}
	return r
}

func (r *redirect) withLabel(label string) *redirect {
	r.RedirectLabel = label
	return r
}

// withContinue adds the continue_with the frontend follows after a
// registration submission.
func (r *redirect) withContinue() *redirect {
	to := r.GetRedirectTo()
	r.ContinueWith = []kClient.ContinueWith{kClient.ContinueWithRedirectBrowserToAsContinueWith(kClient.NewContinueWithRedirectBrowserTo("redirect_browser_to", to))}
	return r
}

func redirectResponse(w http.ResponseWriter, r *http.Request, resp *redirect) {
	w.Header().Set("Content-Type", "application/json")
	kratos.RedirectResponse(w, r, resp)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, id, message string) {
	writeJSON(w, status, kratos.KratosErrorResponse{Error: &kClient.GenericError{Id: &id, Message: message}})
}

func ssoUnavailable(w http.ResponseWriter) {
	writeError(w, http.StatusServiceUnavailable, ssoUnavailableError, ssoUnavailableMessage)
}

// tenantsUnavailable answers an outage of tenant-service: the user may try
// again in a moment. It carries the error ID the frontend shows in place for
// an outage; the message tells the two apart.
func tenantsUnavailable(w http.ResponseWriter) {
	writeError(w, http.StatusServiceUnavailable, ssoUnavailableError, tenantsUnavailableMessage)
}

func ssoNotApplicable(w http.ResponseWriter) {
	writeError(w, http.StatusForbidden, ssoNotApplicableError, ssoNotApplicableMessage)
}

// isUnavailable reports whether a call to tenant-service or sso-service
// failed because the service could not be reached or did not answer in time,
// the request budget running out included: the same call may work in a
// moment. Any other error is the answer of the service.
func isUnavailable(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	switch status.Code(err) {
	case codes.Unavailable, codes.DeadlineExceeded:
		return true
	}
	return false
}

// budget returns r under requestBudget. The request it was given is left as
// it was, for the handler that goes on with it.
func budget(r *http.Request) (*http.Request, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(r.Context(), requestBudget)
	return r.WithContext(ctx), cancel
}

// peekRequestBody reads the body and puts it back for the next reader.
func peekRequestBody(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	r.Body = io.NopCloser(bytes.NewReader(raw))
	return raw, nil
}

// errDuplicateField is returned for a submission that names a field twice.
var errDuplicateField = errors.New("a field is named twice")

// requestFields returns the top-level fields of a submission, from its query
// and its JSON or form body, leaving the body readable. The handler decodes
// the same body into a struct, whose fields encoding/json matches whatever
// their case, the last of two winning: names are folded as it folds them, and
// a field named twice is an error.
func requestFields(r *http.Request) (map[string]any, error) {
	fields := map[string]any{}
	for k, v := range r.URL.Query() {
		if err := addField(fields, k, v[0]); err != nil {
			return nil, err
		}
	}
	raw, err := peekRequestBody(r)
	if err != nil || len(raw) == 0 {
		return fields, nil
	}
	body, err := jsonFields(raw)
	if errors.Is(err, errDuplicateField) {
		return nil, err
	}
	if err != nil {
		body = map[string]any{}
		if values, err := url.ParseQuery(string(raw)); err == nil {
			for k, v := range values {
				if err := addField(body, k, v[0]); err != nil {
					return nil, err
				}
			}
		}
	}
	maps.Copy(fields, body)
	return fields, nil
}

// jsonFields returns the fields of the JSON object the body starts with,
// which is what the json.Decoder of the handler reads from it.
func jsonFields(raw []byte) (map[string]any, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	if start, err := dec.Token(); err != nil || start != json.Delim('{') {
		return nil, errors.New("not a JSON object")
	}
	fields := map[string]any{}
	for dec.More() {
		token, err := dec.Token()
		if err != nil {
			return nil, err
		}
		name, _ := token.(string)
		var value any
		if err := dec.Decode(&value); err != nil {
			return nil, err
		}
		if err := addField(fields, name, value); err != nil {
			return nil, err
		}
	}
	return fields, nil
}

func addField(fields map[string]any, name string, value any) error {
	name = foldName(name)
	if _, ok := fields[name]; ok {
		return errDuplicateField
	}
	fields[name] = value
	return nil
}

// foldName folds a field name as encoding/json folds it to match the field
// of a struct, with ASCII letters in lower case.
func foldName(name string) string {
	return strings.Map(func(r rune) rune {
		// the smallest of the runes that fold to one another
		for {
			next := unicode.SimpleFold(r)
			if next <= r {
				r = next
				break
			}
			r = next
		}
		if 'A' <= r && r <= 'Z' {
			r += 'a' - 'A'
		}
		return r
	}, name)
}

func stringField(fields map[string]any, name string) string {
	s, _ := fields[name].(string)
	return s
}

// traitsEmail returns the email trait of a registration submission, nested
// ("traits": {"email"}) or flat ("traits.email"). It returns false when the
// two differ: which one the handler reads depends on the method.
func traitsEmail(fields map[string]any) (string, bool) {
	flat, hasFlat := fields["traits.email"].(string)
	if traits, ok := fields["traits"].(map[string]any); ok {
		if nested, ok := traits["email"].(string); ok {
			return nested, !hasFlat || nested == flat
		}
	}
	return flat, true
}

// isFirstFactorSubmission reports whether a submission on an aal1 flow
// completes or starts a first factor. An oidc submission may come without a
// method.
func isFirstFactorSubmission(fields map[string]any) bool {
	switch stringField(fields, "method") {
	case methodPassword, methodOIDC, methodPasskey, methodWebAuthn, methodCode:
		return true
	}
	return stringField(fields, "provider") != ""
}

// sameOrigin reports whether a request comes from the pages of login-ui: its
// Origin, which browsers send with a POST, is that of login-ui; without one,
// Sec-Fetch-Site says that it does not come from another site. A request
// with neither does not.
func sameOrigin(r *http.Request, origin string) bool {
	if o := r.Header.Get("Origin"); o != "" {
		return strings.EqualFold(o, origin)
	}
	switch r.Header.Get("Sec-Fetch-Site") {
	case "same-origin", "none":
		return true
	}
	return false
}

// emailFromSession returns the email trait of the identity, or "".
func emailFromSession(session *kClient.Session) string {
	if session == nil || session.Identity == nil {
		return ""
	}
	traits, ok := session.Identity.Traits.(map[string]interface{})
	if !ok {
		return ""
	}
	s, _ := traits["email"].(string)
	return s
}

// addressUnverified reports whether the email trait of the session's identity
// is one of its verifiable addresses and is not verified.
func addressUnverified(session *kClient.Session) bool {
	email := emailFromSession(session)
	if email == "" {
		return false
	}
	for _, a := range session.Identity.VerifiableAddresses {
		if strings.EqualFold(a.Value, email) {
			return !a.Verified
		}
	}
	return false
}

// withoutSession drops the Kratos session cookie: a company sign-in, or a
// fresh first factor, never carries a session into Kratos.
func withoutSession(c []*http.Cookie) []*http.Cookie {
	return httpHelpers.FilterCookies(c, kratos.KRATOS_SESSION_COOKIE_NAME)
}

// mergeCookies overlays extra, the cookies Kratos just set, on base.
func mergeCookies(base, extra []*http.Cookie) []*http.Cookie {
	if len(extra) == 0 {
		return base
	}
	out := make([]*http.Cookie, 0, len(base)+len(extra))
	names := map[string]bool{}
	for _, c := range extra {
		names[c.Name] = true
	}
	for _, c := range base {
		if !names[c.Name] {
			out = append(out, c)
		}
	}
	return append(out, extra...)
}

func setCookies(w http.ResponseWriter, c []*http.Cookie) {
	for _, cookie := range c {
		http.SetCookie(w, cookie)
	}
}

// challengeFromReturnTo reads the login challenge of the URL a flow returns
// to.
func challengeFromReturnTo(returnTo string) string {
	u, err := url.Parse(returnTo)
	if err != nil {
		return ""
	}
	return u.Query().Get("login_challenge")
}

// loginURL returns the login page of a Hydra login request, where every
// company sign-in for one returns.
func (a *API) loginURL(loginChallenge string) string {
	u, _ := url.JoinPath(a.baseURL, "/ui/login")
	if loginChallenge == "" {
		return u
	}
	return u + "?" + url.Values{"login_challenge": {loginChallenge}}.Encode()
}

// completeURL returns the completePath URL going on to returnTo.
func (a *API) completeURL(returnTo string) string {
	u, _ := url.JoinPath(a.baseURL, completePath)
	if returnTo == "" {
		return u
	}
	return u + "?" + url.Values{"return_to": {returnTo}}.Encode()
}

// loginFlowURL returns the login page of a flow.
func (a *API) loginFlowURL(flowID string) string {
	return a.pageURL("/ui/login", url.Values{"flow": {flowID}})
}

// pageURL returns a same-origin page under the context path.
func (a *API) pageURL(page string, q url.Values) string {
	p, _ := url.JoinPath("/", a.contextPath, page)
	if len(q) == 0 {
		return p
	}
	return p + "?" + q.Encode()
}

func (a *API) absoluteURL(page string) string {
	u, _ := url.JoinPath(a.baseURL, page)
	return u
}
