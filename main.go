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
	mdClear := cfg.middlewareCheckClearance

	mux.Handle("/app/", mdMetrics(http.StripPrefix("/app", http.FileServer(http.Dir(".")))))

	// REGISTER API HANDLERS
	mux.HandleFunc("GET /api/healthz", endpReadiness)
	mux.HandleFunc("GET /admin/metrics", cfg.endpFileserverHitCountGet)
	mux.HandleFunc("POST /admin/reset", cfg.endpDeleteAllUsers)
	  // User authentication
	mux.HandleFunc("POST /api/users", cfg.endpCreateUser)
	mux.HandleFunc("DELETE /api/users", mdAuth(cfg.endpDeleteUser))
	mux.HandleFunc("PUT /api/users", mdAuth(cfg.endpUpdateUserCredentials))
	mux.HandleFunc("POST /api/login", cfg.endpLoginUser)
	mux.HandleFunc("POST /api/refresh", cfg.endpCheckRefreshToken)
	mux.HandleFunc("POST /api/revoke", cfg.endpRevokeRefreshToken)
	  // Budget setup
	mux.HandleFunc("POST /api/budgets", mdAuth(cfg.endpCreateBudget))
	mux.HandleFunc("POST /api/budgets/{budget_id}/members", mdAuth(mdClear(MANAGER, cfg.endpAddBudgetMemberWithRole)))
	mux.HandleFunc("DELETE /api/budgets/{budget_id}", mdAuth(mdClear(ADMIN, cfg.endpDeleteBudget)))
	mux.HandleFunc("DELETE /api/budgets/{budget_id}/members/{user_id}", mdAuth(mdClear(MANAGER, cfg.endpRemoveBudgetMember)))
	mux.HandleFunc("GET /api/budgets", mdAuth(cfg.endpGetUserBudgets))
	mux.HandleFunc("GET /api/budgets/{budget_id}", mdAuth(mdClear(VIEWER, cfg.endpGetBudget)))
	  // Groups & Categories
	mux.HandleFunc("POST /api/budgets/{budget_id}/groups", mdAuth(mdClear(MANAGER, cfg.endpCreateGroup)))
	mux.HandleFunc("POST /api/budgets/{budget_id}/categories", mdAuth(mdClear(MANAGER, cfg.endpCreateCategory)))
	mux.HandleFunc("GET /api/budgets/{budget_id}/groups", mdAuth(mdClear(VIEWER, cfg.endpGetGroups)))
	mux.HandleFunc("GET /api/budgets/{budget_id}/categories", mdAuth(mdClear(VIEWER, cfg.endpGetCategories)))
	mux.HandleFunc("PUT /api/budgets/{budget_id}/categories/{category_id}", mdAuth(mdClear(MANAGER, cfg.endpAssignCategoryToGroup)))
	mux.HandleFunc("DELETE /api/budgets/{budget_id}/groups/{group_id}", mdAuth(mdClear(MANAGER, cfg.endpDeleteGroup)))
	mux.HandleFunc("DELETE /api/budgets/{budget_id}/categories/{category_id}", mdAuth(mdClear(MANAGER, cfg.endpDeleteCategory)))
		// Payees
	mux.HandleFunc("POST /api/budgets/{budget_id}/payees", mdAuth(mdClear(CONTRIBUTOR, cfg.endpCreatePayee)))
	mux.HandleFunc("GET /api/budgets/{budget_id}/payees", mdAuth(mdClear(VIEWER, cfg.endpGetPayees)))
	mux.HandleFunc("GET /api/budgets/{budget_id}/payees/{payee_id}", mdAuth(mdClear(VIEWER, cfg.endpGetPayee)))
	mux.HandleFunc("DELETE /api/budgets/{budget_id}/payees/{payee_id}", mdAuth(mdClear(CONTRIBUTOR, cfg.endpDeletePayee)))
		// Accounts
	mux.HandleFunc("POST /api/budgets/{budget_id}/accounts", mdAuth(mdClear(MANAGER, cfg.endpAddAccount)))
	mux.HandleFunc("GET /api/budgets/{budget_id}/accounts", mdAuth(mdClear(VIEWER, cfg.endpGetAccounts)))
	mux.HandleFunc("GET /api/budgets/{budget_id}/account/{account_id}", mdAuth(mdClear(VIEWER, cfg.endpGetAccount)))
	mux.HandleFunc("DELETE /api/budgets/{budget_id}/accounts/{account_id}", mdAuth(mdClear(CONTRIBUTOR, cfg.endpDeleteAccount)))
		// Transactions
	mux.HandleFunc("POST /api/budgets/{budget_id}/transactions", mdAuth(mdClear(CONTRIBUTOR, cfg.endpLogTransaction)))
	mux.HandleFunc("GET /api/budgets/{budget_id}/transactions", mdAuth(mdClear(VIEWER, cfg.endpGetTransactions)))
	mux.HandleFunc("GET /api/budgets/{budget_id}/transactions/{transaction_id}", mdAuth(mdClear(VIEWER, cfg.endpGetTransaction)))
	mux.HandleFunc("DELETE /api/budgets/{budget_id}/transactions/{transaction_id}", mdAuth(mdClear(CONTRIBUTOR, cfg.endpDeleteTransaction)))

	server := &http.Server{
		Addr:		":" + port,
		Handler:	mux,
	}
	
	// start server
	log.Printf("Serving files from %s on port: %s\n", filepathRoot, port)
	log.Fatal(server.ListenAndServe())
}