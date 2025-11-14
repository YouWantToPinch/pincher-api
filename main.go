package main

import (
	_ "github.com/lib/pq"

	"log"
	"net/http"

	"github.com/YouWantToPinch/pincher-api/internal/api"
)

func main() {
	const filepathRoot = "."

	cfg := api.LoadEnvConfig(".env")

	pincher := &http.Server{
		Handler: api.SetupMux(cfg),
	}

	// start server
	log.Fatal(pincher.ListenAndServe())
}
