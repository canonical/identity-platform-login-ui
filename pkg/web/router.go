package web

import (
	"io/fs"
	"net/http"

	ih "github.com/canonical/identity_platform_login_ui/internal/hydra"
	ik "github.com/canonical/identity_platform_login_ui/internal/kratos"
	"github.com/canonical/identity_platform_login_ui/internal/logging"
	chi "github.com/go-chi/chi/v5"
	middleware "github.com/go-chi/chi/v5/middleware"

	"github.com/canonical/identity_platform_login_ui/pkg/extra"
	"github.com/canonical/identity_platform_login_ui/pkg/kratos"
	"github.com/canonical/identity_platform_login_ui/pkg/prometheus"
	"github.com/canonical/identity_platform_login_ui/pkg/status"
	"github.com/canonical/identity_platform_login_ui/pkg/ui"
)

func NewRouter(kratosClient *ik.Client, hydraClient *ih.Client, distFS fs.FS, logger logging.LoggerInterface) http.Handler {
	router := chi.NewMux()

	router.Use(
		middleware.RequestID,
		middleware.RequestLogger(logging.NewLogFormatter(logger)), // LogFormatter will only work if logger is set to DEBUG level
	)

	kratos.NewAPI(kratosClient, hydraClient, logger).RegisterEndpoints(router)
	extra.NewAPI(kratosClient, hydraClient, logger).RegisterEndpoints(router)
	status.NewAPI(logger).RegisterEndpoints(router)
	ui.NewAPI(distFS, logger).RegisterEndpoints(router)
	prometheus.NewAPI(logger).RegisterEndpoints(router)

	return router
}
