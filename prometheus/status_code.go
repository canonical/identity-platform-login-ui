package prometheus

import "net/http"

type responseWriterWithStatusCode struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriterWithStatusCode) Status() int {
	return rw.statusCode
}

func (rw *responseWriterWithStatusCode) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

func NewResponseWriterWithStatusCode(rw http.ResponseWriter) *responseWriterWithStatusCode {
	return &responseWriterWithStatusCode{
		rw,
		0,
	}
}

func ResponseWithStatusCodeMiddleware(next http.HandlerFunc) func(rw http.ResponseWriter, r *http.Request) {
	return func(rw http.ResponseWriter, r *http.Request) {
		responseWriter := NewResponseWriterWithStatusCode(rw)
		next.ServeHTTP(responseWriter, r)
	}
}
