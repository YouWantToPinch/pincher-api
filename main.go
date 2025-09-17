package main

import (
	_ "github.com/lib/pq"

	"log"
	"net/http"
	
	"github.com/YouWantToPinch/pincher-api/internal/server"
)

func main() {
	const filepathRoot = "."
	const port = "8080"

	cfg := server.LoadEnvConfig()

	pincher := &http.Server{
		Addr:		":" + port,
		Handler:	server.SetupMux(cfg),
	}
	
	// start server
	log.Printf("Serving files from %s on port: %s\n", filepathRoot, port)
	log.Fatal(pincher.ListenAndServe())
}