// Package api handles routes and their associated handlers
package api

import (
	"net/http"

	_ "github.com/lib/pq"
)

func SetupMux(cfg *APIConfig) *http.ServeMux {
	mux := http.NewServeMux()

	// middleware
	mdAuth := cfg.middlewareAuthenticate
	mdClear := cfg.middlewareCheckClearance

	// REGISTER API HANDLERS
	// ======================

	// Admin & State
	mux.HandleFunc("GET /api/healthz", endpReadiness)
	mux.HandleFunc("POST /admin/reset", cfg.endpDeleteAllUsers)
	mux.HandleFunc("GET /admin/users", cfg.endpGetAllUsers)
	mux.HandleFunc("GET /admin/users/count", cfg.endpGetTotalUserCount)
	// User authentication
	mux.HandleFunc("POST /api/users", cfg.endpCreateUser)
	mux.HandleFunc("DELETE /api/users", mdAuth(cfg.endpDeleteUser))
	mux.HandleFunc("PUT /api/users", mdAuth(cfg.endpUpdateUserCredentials))
	mux.HandleFunc("POST /api/login", cfg.endpLoginUser)
	mux.HandleFunc("POST /api/refresh", cfg.endpCheckRefreshToken)
	mux.HandleFunc("POST /api/revoke", cfg.endpRevokeRefreshToken)
	// Budgets
	mux.HandleFunc("POST /api/budgets", mdAuth(cfg.endpCreateBudget))
	mux.HandleFunc("POST /api/budgets/{budget_id}/members", mdAuth(mdClear(MANAGER, cfg.endpAddBudgetMemberWithRole)))
	mux.HandleFunc("PUT /api/budgets/{budget_id}", mdAuth(mdClear(MANAGER, cfg.endpUpdateBudget)))
	mux.HandleFunc("DELETE /api/budgets/{budget_id}", mdAuth(mdClear(ADMIN, cfg.endpDeleteBudget)))
	mux.HandleFunc("DELETE /api/budgets/{budget_id}/members/{user_id}", mdAuth(mdClear(MANAGER, cfg.endpRemoveBudgetMember)))
	mux.HandleFunc("GET /api/budgets", mdAuth(cfg.endpGetUserBudgets))
	mux.HandleFunc("GET /api/budgets/{budget_id}", mdAuth(mdClear(VIEWER, cfg.endpGetBudget)))
	mux.HandleFunc("GET /api/budgets/{budget_id}/capital", mdAuth(mdClear(VIEWER, cfg.endpGetBudgetCapital)))
	// Groups & Categories
	mux.HandleFunc("POST /api/budgets/{budget_id}/groups", mdAuth(mdClear(MANAGER, cfg.endpCreateGroup)))
	mux.HandleFunc("POST /api/budgets/{budget_id}/categories", mdAuth(mdClear(MANAGER, cfg.endpCreateCategory)))
	mux.HandleFunc("GET /api/budgets/{budget_id}/groups", mdAuth(mdClear(VIEWER, cfg.endpGetGroups)))
	mux.HandleFunc("GET /api/budgets/{budget_id}/categories", mdAuth(mdClear(VIEWER, cfg.endpGetCategories)))
	mux.HandleFunc("PUT /api/budgets/{budget_id}/groups/{group_id}", mdAuth(mdClear(MANAGER, cfg.endpUpdateGroup)))
	mux.HandleFunc("PUT /api/budgets/{budget_id}/categories/{category_id}", mdAuth(mdClear(MANAGER, cfg.endpUpdateCategory)))
	mux.HandleFunc("DELETE /api/budgets/{budget_id}/groups/{group_id}", mdAuth(mdClear(MANAGER, cfg.endpDeleteGroup)))
	mux.HandleFunc("DELETE /api/budgets/{budget_id}/categories/{category_id}", mdAuth(mdClear(MANAGER, cfg.endpDeleteCategory)))
	// Payees
	mux.HandleFunc("POST /api/budgets/{budget_id}/payees", mdAuth(mdClear(CONTRIBUTOR, cfg.endpCreatePayee)))
	mux.HandleFunc("GET /api/budgets/{budget_id}/payees", mdAuth(mdClear(VIEWER, cfg.endpGetPayees)))
	mux.HandleFunc("GET /api/budgets/{budget_id}/payees/{payee_id}", mdAuth(mdClear(VIEWER, cfg.endpGetPayee)))
	mux.HandleFunc("PUT /api/budgets/{budget_id}/payees/{payee_id}", mdAuth(mdClear(MANAGER, cfg.endpUpdatePayee)))
	mux.HandleFunc("DELETE /api/budgets/{budget_id}/payees/{payee_id}", mdAuth(mdClear(CONTRIBUTOR, cfg.endpDeletePayee)))
	// Accounts
	mux.HandleFunc("POST /api/budgets/{budget_id}/accounts", mdAuth(mdClear(MANAGER, cfg.endpAddAccount)))
	mux.HandleFunc("GET /api/budgets/{budget_id}/accounts", mdAuth(mdClear(VIEWER, cfg.endpGetAccounts)))
	mux.HandleFunc("GET /api/budgets/{budget_id}/accounts/{account_id}", mdAuth(mdClear(VIEWER, cfg.endpGetAccount)))
	mux.HandleFunc("GET /api/budgets/{budget_id}/accounts/{account_id}/capital", mdAuth(mdClear(VIEWER, cfg.endpGetBudgetAccountCapital)))
	mux.HandleFunc("PUT /api/budgets/{budget_id}/accounts/{account_id}", mdAuth(mdClear(MANAGER, cfg.endpUpdateAccount)))
	mux.HandleFunc("PATCH /api/budgets/{budget_id}/accounts/{account_id}", mdAuth(mdClear(MANAGER, cfg.endpRestoreAccount)))
	mux.HandleFunc("DELETE /api/budgets/{budget_id}/accounts/{account_id}", mdAuth(mdClear(CONTRIBUTOR, cfg.endpDeleteAccount)))
	// Transactions
	mux.HandleFunc("POST /api/budgets/{budget_id}/transactions", mdAuth(mdClear(CONTRIBUTOR, cfg.endpLogTransaction)))
	mux.HandleFunc("GET /api/budgets/{budget_id}/accounts/{account_id}/transactions", mdAuth(mdClear(VIEWER, cfg.endpGetTransactions)))
	mux.HandleFunc("GET /api/budgets/{budget_id}/categories/{category_id}/transactions", mdAuth(mdClear(VIEWER, cfg.endpGetTransactions)))
	mux.HandleFunc("GET /api/budgets/{budget_id}/payees/{payee_id}/transactions", mdAuth(mdClear(VIEWER, cfg.endpGetTransactions)))
	mux.HandleFunc("GET /api/budgets/{budget_id}/transactions", mdAuth(mdClear(VIEWER, cfg.endpGetTransactions)))
	mux.HandleFunc("GET /api/budgets/{budget_id}/transactions/{transaction_id}", mdAuth(mdClear(VIEWER, cfg.endpGetTransaction)))
	mux.HandleFunc("GET /api/budgets/{budget_id}/transactions/{transaction_id}/splits", mdAuth(mdClear(VIEWER, cfg.endpGetTransactionSplits)))
	mux.HandleFunc("PUT /api/budgets/{budget_id}/transactions/{transaction_id}", mdAuth(mdClear(MANAGER, cfg.endpUpdateTransaction)))
	mux.HandleFunc("DELETE /api/budgets/{budget_id}/transactions/{transaction_id}", mdAuth(mdClear(CONTRIBUTOR, cfg.endpDeleteTransaction)))
	// Months & Dollar Assignment
	mux.HandleFunc("POST /api/budgets/{budget_id}/months/{month_id}/categories/{category_id}", mdAuth(mdClear(MANAGER, cfg.endpAssignAmountToCategory)))
	mux.HandleFunc("GET /api/budgets/{budget_id}/months/{month_id}/categories/{category_id}", mdAuth(mdClear(MANAGER, cfg.endpGetMonthCategoryReport)))
	mux.HandleFunc("GET /api/budgets/{budget_id}/months/{month_id}/categories", mdAuth(mdClear(MANAGER, cfg.endpGetMonthCategories)))
	mux.HandleFunc("GET /api/budgets/{budget_id}/months/{month_id}", mdAuth(mdClear(MANAGER, cfg.endpGetMonthReport)))
	return mux
}
