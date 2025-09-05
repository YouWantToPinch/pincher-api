package main

import (
	_ "github.com/lib/pq"

	"log"
	"os"
	"fmt"
	"database/sql"
	"net/http"

	"github.com/joho/godotenv"
	
	"github.com/YouWantToPinch/pincher-api/internal/database"
)

func main() {
	const filepathRoot = "."
	const port = "8080"

	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	cfg := apiConfig{}
	dbQueries := database.New(db)
	cfg.db = dbQueries

	// Handle any API keys here for webhooks
	/*
	apiKeys := make(map[string]string)
	cfg.apiKeys = &apiKeys
	(*cfg.apiKeys)["api"] = os.Getenv("API_KEY_1")
	*/

	cfg.platform = os.Getenv("PLATFORM")
	cfg.secret = os.Getenv("SECRET")

	mux := http.NewServeMux()

	// middleware
	mdMetrics := cfg.middlewareMetricsInc
	mdAuth := cfg.middlewareAuthenticate

	mux.Handle("/app/", mdMetrics(http.StripPrefix("/app", http.FileServer(http.Dir(".")))))

	// REGISTER API HANDLERS
	mux.HandleFunc("GET /api/healthz", endpReadiness)
	mux.HandleFunc("GET /admin/metrics", cfg.endpFileserverHitCountGet)
	mux.HandleFunc("POST /admin/reset", cfg.endpDeleteAllUsers)
	  // User authentication
	mux.HandleFunc("POST /api/users", cfg.endpCreateUser)
	mux.HandleFunc("DELETE /api/users/{user_id}", mdAuth(cfg.endpDeleteUser))
	mux.HandleFunc("PUT /api/users", mdAuth(cfg.endpUpdateUserCredentials))
	mux.HandleFunc("POST /api/login", cfg.endpLoginUser)
	mux.HandleFunc("POST /api/refresh", cfg.endpCheckRefreshToken)
	mux.HandleFunc("POST /api/revoke", cfg.endpRevokeRefreshToken)
	  // Groups & Categories
	mux.HandleFunc("POST /api/users/{user_id}/groups", mdAuth(cfg.endpCreateGroup))
	mux.HandleFunc("POST /api/users/{user_id}/categories", mdAuth(cfg.endpCreateCategory))
	mux.HandleFunc("GET /api/users/{user_id}/groups", mdAuth(cfg.endpGetGroupsByUserID))
	mux.HandleFunc("GET /api/users/{user_id}/categories", mdAuth(cfg.endpGetCategoriesByUserID))
	mux.HandleFunc("PUT /api/users/{user_id}/categories/{category_id}", mdAuth(cfg.endpAssignCategoryToGroup))
	mux.HandleFunc("DELETE /api/users/{user_id}/groups/{group_id}", mdAuth(cfg.endpDeleteGroup))
	mux.HandleFunc("DELETE /api/users/{user_id}/categories/{category_id}", mdAuth(cfg.endpDeleteCategory))

	server := &http.Server{
		Addr:		":" + port,
		Handler:	mux,
	}
	
	// start server
	log.Printf("Serving files from %s on port: %s\n", filepathRoot, port)
	log.Fatal(server.ListenAndServe())
}