package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
)

const defaultPort = "8080"

//go:embed ui/dist
//go:embed ui/dist/_next
//go:embed ui/dist/_next/static/chunks/pages/*.js
//go:embed ui/dist/_next/static/*/*.js
//go:embed ui/dist/_next/static/*/*.css
var ui embed.FS

func main() {
	dist, _ := fs.Sub(ui, "ui/dist")

	http.Handle("/", http.FileServer(http.FS(dist)))
	http.HandleFunc("/api/kratos/self-service/login/browser", handleCreateFlow)
	http.HandleFunc("/api/kratos/self-service/login", handleUpdateFlow)
	http.HandleFunc("/api/kratos/self-service/errors", handleKratosError)
	http.HandleFunc("/api/consent", handleConsent)

	port := os.Getenv("PORT")

	if port == "" {
		port = defaultPort
	}

	log.Println("Starting server on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleCreateFlow(w http.ResponseWriter, _ *http.Request) {
	// kratos := os.Getenv("KRATOS_PUBLIC_URL")
	return
}

func handleUpdateFlow(w http.ResponseWriter, _ *http.Request) {
	// kratos := os.Getenv("KRATOS_PUBLIC_URL")
	return
}

func handleKratosError(w http.ResponseWriter, _ *http.Request) {
	// kratos := os.Getenv("KRATOS_PUBLIC_URL")
	return
}

func handleConsent(w http.ResponseWriter, _ *http.Request) {
	// kratos := os.Getenv("KRATOS_PUBLIC_URL")
	// hydra := os.Getenv("HYDRA_ADMIN_URL")
	return
}
