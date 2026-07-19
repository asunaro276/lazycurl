// Command mockserver is a small, dependency-free HTTP server used to
// exercise lazycurl's curl argv construction (internal/curlexec) and TUI
// end-to-end, both from automated tests (via testcontainers-go) and from a
// developer's terminal (via docker compose). It has no relationship to the
// lazycurl root Go module beyond communicating over plain HTTP.
package main

import (
	"flag"
	"log"
	"net/http"
	"os"
)

func main() {
	port := flag.String("port", envOr("PORT", "8080"), "port to listen on")
	flag.Parse()

	log.Printf("mockserver listening on :%s", *port)
	if err := http.ListenAndServe(":"+*port, newMux()); err != nil {
		log.Fatal(err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// newMux registers all mockserver endpoints on a fresh ServeMux.
func newMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/echo", handleEcho)
	mux.HandleFunc("/status/{code}", handleStatus)
	mux.HandleFunc("/redirect/{n}", handleRedirect)
	mux.HandleFunc("/delay/{sec}", handleDelay)
	mux.HandleFunc("/stream", handleStream)
	mux.HandleFunc("/auth/basic", handleAuthBasic)
	mux.HandleFunc("/auth/bearer", handleAuthBearer)
	return mux
}
