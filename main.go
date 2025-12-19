package main

import (
	"embed"
	"fmt"
	"log"
	"log/slog"
	"net/http"

	_ "github.com/lib/pq"

	"github.com/YouWantToPinch/pincher-api/internal/api"
)

//go:embed sql/schema/*.sql
var embedMigrations embed.FS

func main() {
	const port = "8080"

	cfg := &api.APIConfig{}
	cfg.Init(".env", "")
	cfg.ConnectToDB(embedMigrations, "sql/schema")

	pincher := &http.Server{
		Addr:    ":" + port,
		Handler: api.SetupMux(cfg),
	}

	url := "http://localhost" + pincher.Addr
	slog.Info(fmt.Sprintf("Server listening at: %s", url))

	// start server
	log.Fatal(pincher.ListenAndServe())
}
