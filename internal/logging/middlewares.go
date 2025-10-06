// Copyright 2024 Canonical Ltd
// SPDX-License-Identifier: AGPL

package logging

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

const (
	UserAgentKey     = "useragent"
	SourceIpKey      = "source_ip"
	HostnameKey      = "hostname"
	ProtocolKey      = "protocol"
	PortKey          = "port"
	RequestUriKey    = "request_uri"
	RequestMethodKey = "request_method"
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

func LogContextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		ctx = context.WithValue(ctx, UserAgentKey, r.UserAgent())
		ctx = context.WithValue(ctx, SourceIpKey, r.RemoteAddr)
		ctx = context.WithValue(ctx, ProtocolKey, r.Proto)
		ctx = context.WithValue(ctx, RequestUriKey, r.RequestURI)
		ctx = context.WithValue(ctx, RequestMethodKey, r.Method)

		h := strings.Split(r.Host, ":")
		ctx = context.WithValue(ctx, HostnameKey, h[0])
		if len(h) > 1 {
			ctx = context.WithValue(ctx, PortKey, h[1])
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
