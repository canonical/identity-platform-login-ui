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
	"path"
	"regexp"

	"syscall"
	"time"

	prometheus "github.com/canonical/identity_platform_login_ui/prometheus"

	"github.com/canonical/identity_platform_login_ui/pkg/extra"
	"github.com/canonical/identity_platform_login_ui/pkg/kratos"

	"github.com/canonical/identity_platform_login_ui/pkg/status"

	ih "github.com/canonical/identity_platform_login_ui/internal/hydra"
	ik "github.com/canonical/identity_platform_login_ui/internal/kratos"
)

const defaultPort = "8080"

var oidcScopeMapping = map[string][]string{
	"openid": {"sub"},
	"profile": {
		"name",
		"family_name",
		"given_name",
		"middle_name",
		"nickname",
		"preferred_username",
		"profile",
		"picture",
		"website",
		"gender",
		"birthdate",
		"zoneinfo",
		"locale",
		"updated_at",
	},
	"email":   {"email", "email_verified"},
	"address": {"address"},
	"phone":   {"phone_number", "phone_number_verified"},
}

//go:embed ui/dist
//go:embed ui/dist/_next
//go:embed ui/dist/_next/static/chunks/pages/*.js
//go:embed ui/dist/_next/static/*/*.js
//go:embed ui/dist/_next/static/*/*.css
var ui embed.FS

func main() {
	metricsManager := setUpPrometheus()

	dist, _ := fs.Sub(ui, "ui/dist")
	fs := http.FileServer(http.FS(dist))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Add the html suffix if missing
		// This allows us to serve /login.html in the /login URL
		if ext := path.Ext(r.URL.Path); ext == "" && r.URL.Path != "/" {
			r.URL.Path += ".html"
		}
		metricsManager.Middleware(fs.ServeHTTP)(w, r)
	})

	kClient := ik.NewClient(os.Getenv("KRATOS_PUBLIC_URL"))
	hClient := ih.NewClient(os.Getenv("HYDRA_ADMIN_URL"))

	kratos.NewAPI(kClient, hClient).RegisterEndpoints(http.DefaultServeMux)
	extra.NewAPI(kClient, hClient).RegisterEndpoints(http.DefaultServeMux)
	status.NewAPI().RegisterEndpoints(http.DefaultServeMux)

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

func setUpPrometheus() *prometheus.MetricsManager {
	mm := prometheus.NewMetricsManagerWithPrefix("identity-platform-login-ui-operator", "http", "", "", "")
	mm.RegisterRoutes(
		"/api/kratos/self-service/login/browser",
		"/api/kratos/self-service/login/flows",
		"/api/kratos/self-service/login",
		"/api/kratos/self-service/errors",
		"/api/consent",
		"/health/alive",
		prometheus.PrometheusPath,
	)

	pages, err := ui.ReadDir("ui/dist")
	if err != nil {
		log.Printf("Error when calling `setUpPrometheus`: %v\n", err)
	}
	mm.RegisterRoutes(registerHelper(pages...)...)
	return mm
}

func registerHelper(dirs ...fs.DirEntry) []string {
	r, _ := regexp.Compile("html")
	ret := make([]string, 0)
	for _, d := range dirs {
		name := d.Name()
		if r.MatchString(name) {
			ret = append(ret, name)
		}
	}
	ret = append(ret, "/")

	return ret
}
