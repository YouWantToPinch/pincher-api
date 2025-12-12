package main

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"

	_ "github.com/lib/pq"

	"github.com/YouWantToPinch/pincher-api/internal/api"
)

func main() {
	const port = "8080"

	cfg := api.LoadEnvConfig(".env")

	pincher := &http.Server{
		Addr:    ":" + port,
		Handler: api.SetupMux(cfg),
	}

	url := "http://localhost" + pincher.Addr
	slog.Info(fmt.Sprintf("Server listening at: %s", url))

	// start server
	log.Fatal(pincher.ListenAndServe())
}
