package logging

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// brain-picked from DefaultLogFormatter https://raw.githubusercontent.com/go-chi/chi/v5.0.8/middleware/logger.go

// LogFormatter is a simple logger that implements a middleware.LogFormatter.
type LogFormatter struct {
	Logger LoggerInterface
}

// NewLogEntry creates a new LogEntry for the request.
func (l *LogFormatter) NewLogEntry(r *http.Request) middleware.LogEntry {
	entry := new(LogEntry)

	entry.LogFormatter = l
	entry.request = r
	entry.buf = new(bytes.Buffer)

	reqID := middleware.GetReqID(r.Context())
	if reqID != "" {
		fmt.Fprintf(entry.buf, "[%s] ", reqID)
	}

	fmt.Fprintf(entry.buf, "%s ", r.Method)

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}

	fmt.Fprintf(entry.buf, "%s://%s%s %s ", scheme, r.Host, r.RequestURI, r.Proto)
	fmt.Fprintf(entry.buf, "from %s ", r.RemoteAddr)

	return entry
}

type LogEntry struct {
	*LogFormatter
	request *http.Request
	buf     *bytes.Buffer
}

func (l *LogEntry) Write(status, bytes int, header http.Header, elapsed time.Duration, extra interface{}) {

	fmt.Fprintf(l.buf, "%v %03d %dB in %s", header, status, bytes, elapsed)

	l.Logger.Debug(l.buf.String())
}

// TODO @shipperizer see if implementing this or not
func (l *LogEntry) Panic(v interface{}, stack []byte) {
	return
}

func NewLogFormatter(logger LoggerInterface) *LogFormatter {
	l := new(LogFormatter)

	l.Logger = logger

	return l
}
