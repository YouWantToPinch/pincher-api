package main

import (
	"fmt"
	_ "github.com/lib/pq"
	"log"
	"log/slog"
	"net/http"

	"github.com/YouWantToPinch/pincher-api/internal/api"
)

func main() {
	const filepathRoot = "."
	const port = "8080"

	cfg := api.LoadEnvConfig(".env")

	pincher := &http.Server{
		Addr:    ":" + port,
		Handler: api.SetupMux(cfg),
	}

	addr := ":8080"
	url := "http://localhost" + addr
	slog.Info(fmt.Sprintf("Server listening at: %s", url))

	// start server
	log.Fatal(pincher.ListenAndServe())
}
