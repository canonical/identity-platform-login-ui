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

	pMonitor "github.com/canonical/identity_platform_login_ui/internal/monitoring/prometheus"
	"github.com/canonical/identity_platform_login_ui/pkg/web"

	ih "github.com/canonical/identity_platform_login_ui/internal/hydra"
	ik "github.com/canonical/identity_platform_login_ui/internal/kratos"
	"github.com/canonical/identity_platform_login_ui/internal/logging"
)

const defaultPort = "8080"

//go:embed ui/dist
//go:embed ui/dist/_next
//go:embed ui/dist/_next/static/chunks/pages/*.js
//go:embed ui/dist/_next/static/*/*.js
//go:embed ui/dist/_next/static/*/*.css
var jsFS embed.FS

func main() {
	logger := logging.NewLogger(os.Getenv("LOG_LEVEL"))

	monitor := pMonitor.NewMonitor("identity-login-ui", logger)

	distFS, err := fs.Sub(jsFS, "ui/dist")

	if err != nil {
		log.Fatalf("issue with js distribution files %s", err)
	}

	kClient := ik.NewClient(os.Getenv("KRATOS_PUBLIC_URL"))
	hClient := ih.NewClient(os.Getenv("HYDRA_ADMIN_URL"))

	router := web.NewRouter(kClient, hClient, distFS, monitor, logger)

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
		Handler:      router,
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

	logger.Desugar().Sync()

	// Optionally, you could run srv.Shutdown in a goroutine and block on
	// <-ctx.Done() if your application should wait for other services
	// to finalize based on context cancellation.
	log.Println("Shutting down")
	os.Exit(0)

}
