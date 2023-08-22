package monitoring

import (
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/canonical/identity-platform-login-ui/internal/logging"
	"github.com/go-chi/chi/v5/middleware"
)

const (
	// IDPathRegex regexp used to swap the {id*} parameters in the path with simply id
	// supports alphabetic characters and underscores, no dashes
	IDPathRegex string = "{[a-zA-Z_]*}"
)

// Middleware is the monitoring middleware object implementing Prometheus monitoring
type Middleware struct {
	service string
	regex   *regexp.Regexp

	monitor MonitorInterface
	logger  logging.LoggerInterface
}

func (mdw *Middleware) ResponseTime() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
				startTime := time.Now()

				next.ServeHTTP(ww, r)

				tags := map[string]string{
					"route":  fmt.Sprintf("%s%s", r.Method, mdw.regex.ReplaceAll([]byte(r.URL.Path), []byte("id"))),
					"status": fmt.Sprint(ww.Status()),
				}

				mdw.monitor.SetResponseTimeMetric(tags, time.Since(startTime).Seconds())
			},
		)
	}
}

// NewMiddleware returns a Middleware based on the type of monitor
func NewMiddleware(monitor MonitorInterface, logger logging.LoggerInterface) *Middleware {
	mdw := new(Middleware)

	mdw.monitor = monitor

	mdw.service = monitor.GetService()
	mdw.logger = logger
	mdw.regex = regexp.MustCompile(IDPathRegex)

	return mdw
}
