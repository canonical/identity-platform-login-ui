package http_meta

import (
	"net/http"
	"sync/atomic"
)

type responseWriterMeta struct {
	http.ResponseWriter
	statusCode int
	size       int32
}

func (rw *responseWriterMeta) Status() int {
	return rw.statusCode
}

func (rw *responseWriterMeta) Size() int32 {
	return atomic.LoadInt32(&rw.size)
}

func (rw *responseWriterMeta) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

func (rw *responseWriterMeta) Write(p []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(p)
	atomic.AddInt32(&rw.size, int32(n))
	return n, err
}

func NewresponseWriterMeta(rw http.ResponseWriter) *responseWriterMeta {
	return &responseWriterMeta{
		rw,
		0,
		0,
	}
}

func ResponseWriterMetaMiddleware(next http.HandlerFunc) func(rw http.ResponseWriter, r *http.Request) {
	return func(rw http.ResponseWriter, r *http.Request) {
		responseWriter := NewresponseWriterMeta(rw)
		next.ServeHTTP(responseWriter, r)
	}
}

// Unpacks response status from ResponseWriter
func GetResponseStatus(w http.ResponseWriter) (status int) {
	switch t := w.(type) {
	case interface{ Status() int }:
		status = t.Status()
	}

	return
}

// Unpacks response size from ResponseWriter
func GetResponseSize(w http.ResponseWriter) (size int) {
	switch t := w.(type) {
	case interface{ Size() int32 }:
		size = int(t.Size())

	}

	return
}
