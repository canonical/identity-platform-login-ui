package http_meta

import "net/http"

type RestInterface interface {
	Get(string, http.HandlerFunc)
	Put(string, http.HandlerFunc)
	Post(string, http.HandlerFunc)
	Delete(string, http.HandlerFunc)
	HandleFunc(string, http.HandlerFunc)
}
