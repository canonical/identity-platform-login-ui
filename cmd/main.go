package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"

	"syscall"
	"time"

	"github.com/canonical/identity_platform_login_ui/pkg/extra"
	"github.com/canonical/identity_platform_login_ui/pkg/kratos"
	"github.com/canonical/identity_platform_login_ui/pkg/prometheus"
	"github.com/canonical/identity_platform_login_ui/pkg/status"
	"github.com/canonical/identity_platform_login_ui/pkg/ui"

	ih "github.com/canonical/identity_platform_login_ui/internal/hydra"
	ik "github.com/canonical/identity_platform_login_ui/internal/kratos"
)

const defaultPort = "8080"

//go:embed ui/dist
//go:embed ui/dist/_next
//go:embed ui/dist/_next/static/chunks/pages/*.js
//go:embed ui/dist/_next/static/*/*.js
//go:embed ui/dist/_next/static/*/*.css
var jsFS embed.FS

func main() {

	distFS, err := fs.Sub(jsFS, "ui/dist")

	if err != nil {
		log.Fatalf("issue with js distribution files %s", err)
	}

	kClient := ik.NewClient(os.Getenv("KRATOS_PUBLIC_URL"))
	hClient := ih.NewClient(os.Getenv("HYDRA_ADMIN_URL"))

	kratos.NewAPI(kClient, hClient).RegisterEndpoints(http.DefaultServeMux)
	extra.NewAPI(kClient, hClient).RegisterEndpoints(http.DefaultServeMux)
	status.NewAPI().RegisterEndpoints(http.DefaultServeMux)
	ui.NewAPI(distFS).RegisterEndpoints(http.DefaultServeMux)
	prometheus.NewAPI().RegisterEndpoints(http.DefaultServeMux)

	prometheus.NewMetricsManagerWithPrefix(
		"identity-platform-login-ui-operator",
		"http",
		"",
		"",
		"",
	).RegisterRoutes(
		"/api/kratos/self-service/login/browser",
		"/api/kratos/self-service/login/flows",
		"/api/kratos/self-service/login",
		"/api/kratos/self-service/errors",
		"/api/consent",
		"/health/alive",
		"/metrics/prometheus",
	)

	port := os.Getenv("PORT")

	if port == "" {
		port = defaultPort
	}

	log.Println("Starting server on port " + port)

	srv := &http.Server{
		Addr:         fmt.Sprintf("0.0.0.0:%s", port),
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      http.DefaultServeMux,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// Block until we receive our signal.
	<-c

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	srv.Shutdown(ctx)
	// Optionally, you could run srv.Shutdown in a goroutine and block on
	// <-ctx.Done() if your application should wait for other services
	// to finalize based on context cancellation.
	log.Println("Shutting down")
	os.Exit(0)

}
