package monitoring

import (
	"fmt"
	"net/http"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/canonical/identity-platform-login-ui/internal/logging"
	"github.com/go-chi/chi/v5/middleware"
)

const (
	// IDPathRegex regexp used to swap the {id*} parameters in the path with simply id
	// supports alphabetic characters and underscores, no dashes
	IDPathRegex string = "{[a-zA-Z_]*}"
	HTML        string = ".html"
	UNKNOWN     string = "unknown"
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
					"route":  mdw.newRouteLabel(r.Method, r.URL.Path),
					"status": fmt.Sprint(ww.Status()),
				}

				m, err := mdw.monitor.GetResponseTimeMetric(tags)

				if err != nil {
					mdw.logger.Debugf("error fetching metric: %s; keep going....", err)

					return
				}

				m.Observe(time.Since(startTime).Seconds())
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

func (mdw *Middleware) newRouteLabel(method, path string) string {
	newPath := stripHTMLExt(mdw.idPathExtractor(path))
	if ok := mdw.monitor.VerifyEndpoint(newPath); !ok {
		newPath = UNKNOWN
	}
	return fmt.Sprintf("%s%s", method, newPath)
}

func (mdw *Middleware) idPathExtractor(path string) string {
	return string(mdw.regex.ReplaceAll([]byte(path), []byte("id")))
}

func stripHTMLExt(urlPath string) string {
	if ext := path.Ext(urlPath); ext == HTML {
		return strings.TrimSuffix(urlPath, HTML)
	}
	return urlPath
}
