package main

import (
	"embed"
	"fmt"
	"log"
	"log/slog"
	"net/http"

	"github.com/YouWantToPinch/pincher-api/internal/api"
)

func shutdown(cfg *api.APIConfig) {
	if cfg.Pool != nil {
		cfg.Pool.Close()
	}
}

//go:embed sql/schema/*.sql
var embedMigrations embed.FS

func main() {
	const port = "8080"

	cfg := &api.APIConfig{}
	defer shutdown(cfg)
	cfg.Init(".env", "")
	cfg.ConnectToDB(embedMigrations, "sql/schema")
	defer cfg.Pool.Close()

	pincher := &http.Server{
		Addr:    ":" + port,
		Handler: api.SetupMux(cfg),
	}

	url := "http://localhost" + pincher.Addr
	slog.Info(fmt.Sprintf("Server listening at: %s", url))

	// start server
	log.Fatal(pincher.ListenAndServe())
}
