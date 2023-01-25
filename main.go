package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
)

const defaultPort = "8080"

var (
	//go:embed ui/dist
	ui embed.FS
)

func main() {
	dist, _ := fs.Sub(ui, "ui/dist")

	http.Handle("/", http.FileServer(http.FS(dist)))
	http.HandleFunc("/api/hydra", handleHydra)
	http.HandleFunc("/api/kratos", handleKratos)

	port := os.Getenv("PORT")

	if port == "" {
		port = defaultPort
	}

	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleHydra(w http.ResponseWriter, _ *http.Request) {
	// hydra := os.Getenv("HYDRA_ADMIN_URL")
	return
}

func handleKratos(w http.ResponseWriter, _ *http.Request) {
	// kratos := os.Getenv("KRATOS_PUBLIC_URL")
	return
}
