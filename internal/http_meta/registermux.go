package http_meta

import (
	"net/http"

	"github.com/canonical/identity-platform-login-ui/internal/monitoring"
	chi "github.com/go-chi/chi/v5"
)

type RegisterMux struct {
	router  *chi.Mux
	monitor monitoring.MonitorInterface
}

func (rmx *RegisterMux) Delete(pattern string, handlerFn http.HandlerFunc) {
	rmx.monitor.RegisterEndpoints(pattern)
	rmx.router.Delete(pattern, handlerFn)
}
func (rmx *RegisterMux) Get(pattern string, handlerFn http.HandlerFunc) {
	rmx.monitor.RegisterEndpoints(pattern)
	rmx.router.Get(pattern, handlerFn)
}
func (rmx *RegisterMux) Post(pattern string, handlerFn http.HandlerFunc) {
	rmx.monitor.RegisterEndpoints(pattern)
	rmx.router.Post(pattern, handlerFn)
}
func (rmx *RegisterMux) Put(pattern string, handlerFn http.HandlerFunc) {
	rmx.monitor.RegisterEndpoints(pattern)
	rmx.router.Put(pattern, handlerFn)
}

func (rmx *RegisterMux) HandleFunc(pattern string, handlerFn http.HandlerFunc) {
	rmx.router.HandleFunc(pattern, handlerFn)
}

func (rmx *RegisterMux) GetMux() *chi.Mux {
	return rmx.router
}

func NewRegisterMux(mux *chi.Mux, monitor monitoring.MonitorInterface) *RegisterMux {
	router := RegisterMux{
		router:  mux,
		monitor: monitor,
	}

	return &router
}
