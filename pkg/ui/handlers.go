package ui

import (
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/canonical/identity-platform-login-ui/internal/logging"
)

const UI = "/ui"

type API struct {
	fileServer http.Handler

	baseURL         string
	csp             string
	kratosPublicURL string

	logger logging.LoggerInterface
}

func (a *API) RegisterEndpoints(mux *chi.Mux) {

	uiHandlerWithHeaders := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set the UI headers
		// Disables the FLoC (Federated Learning of Cohorts) feature on the browser,
		// preventing the current page from being included in the user's FLoC calculation.
		// FLoC is a proposed replacement for third-party cookies to enable interest-based advertising.
		w.Header().Set("Permissions-Policy", "interest-cohort=()")
		// Prevents the browser from trying to guess the MIME type, which can have security implications.
		// This tells the browser to strictly follow the MIME type provided in the Content-Type header.
		w.Header().Set("X-Content-Type-Options", "nosniff")
		// Restricts the page from being displayed in a frame, iframe, or object to avoid click jacking attacks,
		// but allows it if the site is navigating to the same origin.
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		// Sets the Content Security Policy (CSP) for the page, which helps mitigate XSS attacks and data injection attacks.
		// The policy allows loading resources (scripts, styles, images, etc.) only from the same origin ('self'), data URLs, and all subdomains of ubuntu.com.
		w.Header().Set("Content-Security-Policy", a.csp)

		// `no-store`: This will tell any cache system not to cache the index.html file
		// `no-cache`: This will tell any cache system to check if there is a newer version in the server
		// `must-revalidate`: This will tell any cache system to check for newer version of the file
		// this is considered best practice with SPAs
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

		a.uiFiles(w, r)
	})

	// TODO @shipperizer unsure if we deal with any POST/PUT/PATCH via js
	mux.HandleFunc(fmt.Sprintf("%s/*", UI), uiHandlerWithHeaders)

}

// TODO: Validate response when server error handling is implemented
func (a *API) uiFiles(w http.ResponseWriter, r *http.Request) {
	if ext := path.Ext(r.URL.Path); ext == "" {
		r.URL.Path = fmt.Sprintf("%s.html", r.URL.Path)
	}

	r.URL.Path = strings.TrimPrefix(r.URL.Path, UI)

	a.fileServer.ServeHTTP(w, r)
}

func (a *API) getCSP(baseURL, kratosPublicURL string) string {
	b, _ := url.Parse(baseURL)
	k, _ := url.Parse(kratosPublicURL)
	additionalScriptURL := ""
	if k != nil && b != nil && k.Host != b.Host {
		// Allowlist the kratos URL to allow the browser needs to fetch the webauthn.js script
		additionalScriptURL = kratosPublicURL
	}
	return fmt.Sprintf("default-src 'self' data: https://assets.ubuntu.com; script-src 'self' %v; style-src 'self'", additionalScriptURL)
}

func NewAPI(fileSystem fs.FS, baseURL, kratosPublicURL string, logger logging.LoggerInterface) *API {
	a := new(API)

	a.fileServer = http.FileServer(http.FS(fileSystem))

	a.baseURL = baseURL
	a.kratosPublicURL = kratosPublicURL
	a.csp = a.getCSP(baseURL, kratosPublicURL)
	a.logger = logger

	return a
}
