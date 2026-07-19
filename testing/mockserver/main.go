package main

import (
	"log"
	"net/http"
	"os"
)

const defaultPort = "8080"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	addr := ":" + port
	log.Printf("mockserver listening on %s", addr)
	if err := http.ListenAndServe(addr, NewMux()); err != nil {
		log.Fatal(err)
	}
}
