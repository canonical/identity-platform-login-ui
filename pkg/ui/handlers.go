package ui

import (
	"fmt"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/canonical/identity-platform-login-ui/internal/logging"
	"github.com/go-chi/chi/v5"
)

const UI = "/ui"

type API struct {
	fileServer http.Handler
	logger     logging.LoggerInterface
}

func (a *API) RegisterEndpoints(mux *chi.Mux) {
	// TODO @shipperizer unsure if we deal with any POST/PUT/PATCH via js
	mux.HandleFunc(fmt.Sprintf("%s/*", UI), a.uiFiles)

}

// TODO: Validate response when server error handling is implemented
func (a *API) uiFiles(w http.ResponseWriter, r *http.Request) {
	if ext := path.Ext(r.URL.Path); ext == "" {
		r.URL.Path = fmt.Sprintf("%s.html", r.URL.Path)
	}

	r.URL.Path = strings.TrimPrefix(r.URL.Path, UI)

	a.fileServer.ServeHTTP(w, r)
}

func NewAPI(fileSystem fs.FS, logger logging.LoggerInterface) *API {
	a := new(API)

	a.fileServer = http.FileServer(http.FS(fileSystem))

	a.logger = logger

	return a
}
