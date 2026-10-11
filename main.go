package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"gh-storage/internal/github"
	"gh-storage/internal/web"
)

func main() {
	for _, key := range []string{"ME", "GITHUB_TOKEN", "GITHUB_ACCOUNT", "GITHUB_REPO"} {
		if os.Getenv(key) == "" {
			log.Fatalf("%s is not defined", key)
		}
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	gh := github.New(os.Getenv("GITHUB_TOKEN"), os.Getenv("GITHUB_ACCOUNT"), os.Getenv("GITHUB_REPO"))
	server := &http.Server{Addr: ":" + port, Handler: web.New(gh, os.Getenv("ME")), ReadHeaderTimeout: 10 * time.Second}
	log.Printf("version %s, listening on %s, fluid=%s", web.Version, port, os.Getenv("VERCEL_FLUID"))
	log.Fatal(server.ListenAndServe())
}
