package ui

import (
	"io/fs"
	"net/http"
	"path"

	"github.com/canonical/identity_platform_login_ui/internal/logging"
)

type API struct {
	fileServer http.Handler

	logger logging.LoggerInterface
}

func (a *API) RegisterEndpoints(mux *http.ServeMux) {
	mux.HandleFunc("/", a.uiFiles)
}

// TODO: Validate response when server error handling is implemented
func (a *API) uiFiles(w http.ResponseWriter, r *http.Request) {
	if ext := path.Ext(r.URL.Path); ext == "" && r.URL.Path != "/" {
		r.URL.Path += ".html"
	}

	a.fileServer.ServeHTTP(w, r)
}

func NewAPI(fileSystem fs.FS, logger logging.LoggerInterface) *API {
	a := new(API)

	a.fileServer = http.FileServer(http.FS(fileSystem))

	a.logger = logger

	return a
}
