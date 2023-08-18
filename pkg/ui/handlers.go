package ui

import (
	"fmt"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/canonical/identity-platform-login-ui/internal/http_meta"
	"github.com/canonical/identity-platform-login-ui/internal/logging"
	"github.com/canonical/identity-platform-login-ui/internal/monitoring"
)

const UI = "/ui"

var pages []string = []string{
	"consent",
	"error",
	"index",
	"login",
	"oidc_error",
	"registration",
}

type API struct {
	fileServer http.Handler
	logger     logging.LoggerInterface
	monitor    monitoring.MonitorInterface
}

func (a *API) RegisterEndpoints(mux http_meta.RestInterface) {
	// TODO @shipperizer unsure if we deal with any POST/PUT/PATCH via js
	mux.HandleFunc(fmt.Sprintf("%s/*", UI), a.uiFiles)
	for _, endpoint := range pages {
		a.monitor.RegisterEndpoints(fmt.Sprintf("/%s", endpoint))
	}
}

// TODO: Validate response when server error handling is implemented
func (a *API) uiFiles(w http.ResponseWriter, r *http.Request) {
	if ext := path.Ext(r.URL.Path); ext == "" {
		r.URL.Path = fmt.Sprintf("%s.html", r.URL.Path)
	}

	r.URL.Path = strings.TrimPrefix(r.URL.Path, UI)

	a.fileServer.ServeHTTP(w, r)
}

func NewAPI(fileSystem fs.FS, monitor monitoring.MonitorInterface, logger logging.LoggerInterface) *API {
	a := new(API)

	a.fileServer = http.FileServer(http.FS(fileSystem))

	a.monitor = monitor

	a.logger = logger

	return a
}
