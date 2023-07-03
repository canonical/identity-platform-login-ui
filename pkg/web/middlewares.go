package web

import (
	"net/http"

	cors "github.com/go-chi/cors"
)

func middlewareCORS(origins []string) func(http.Handler) http.Handler {
	return cors.Handler(
		cors.Options{
			AllowedOrigins: origins,
			AllowedMethods: []string{
				http.MethodHead,
				http.MethodGet,
				http.MethodPost,
				http.MethodPut,
				http.MethodPatch,
				http.MethodDelete,
				http.MethodOptions,
			},
			AllowedHeaders:   []string{"*"},
			AllowCredentials: true,
			MaxAge:           300, // Maximum value not ignored by any of major browsers
		},
	)
}
