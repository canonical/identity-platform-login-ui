package ui

import (
	"io/fs"
	"net/http"
	"path"
)

type API struct {
	fileServer http.Handler
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

func NewAPI(fileSystem fs.FS) *API {
	a := new(API)

	a.fileServer = http.FileServer(http.FS(fileSystem))

	return a
}
