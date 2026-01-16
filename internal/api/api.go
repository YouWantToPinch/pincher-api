// Package api handles routes and their associated handlers
package api

import (
	"net/http"
)

func SetupMux(cfg *APIConfig) *http.ServeMux {
	mux := http.NewServeMux()

	// middleware
	mdAuth := cfg.middlewareAuthenticate
	mdClear := cfg.middlewareCheckClearance
	mdValidateTxn := cfg.middlewareValidateTxn

	// REGISTER API HANDLERS
	// ======================

	// Admin & State
	mux.HandleFunc("GET /api/healthz", cfg.handleReadiness)
	mux.HandleFunc("POST /admin/reset", cfg.handleDeleteAllUsers)
	mux.HandleFunc("GET /admin/users", cfg.handleGetAllUsers)
	mux.HandleFunc("GET /admin/users/count", cfg.handleGetTotalUserCount)
	// User authentication
	mux.HandleFunc("POST /api/users", cfg.handleCreateUser)
	mux.HandleFunc("DELETE /api/users", mdAuth(cfg.handleDeleteUser))
	mux.HandleFunc("PUT /api/users", mdAuth(cfg.handleUpdateUserCredentials))
	mux.HandleFunc("POST /api/login", cfg.handleLoginUser)
	mux.HandleFunc("POST /api/refresh", cfg.handleCheckRefreshToken)
	mux.HandleFunc("POST /api/revoke", cfg.handleRevokeRefreshToken)
	// Budgets
	mux.HandleFunc("POST /api/budgets", mdAuth(cfg.handleCreateBudget))
	mux.HandleFunc("POST /api/budgets/{budget_id}/members", mdAuth(mdClear(MANAGER, cfg.handleAddBudgetMemberWithRole)))
	mux.HandleFunc("PUT /api/budgets/{budget_id}", mdAuth(mdClear(MANAGER, cfg.handleUpdateBudget)))
	mux.HandleFunc("DELETE /api/budgets/{budget_id}", mdAuth(mdClear(ADMIN, cfg.handleDeleteBudget)))
	mux.HandleFunc("DELETE /api/budgets/{budget_id}/members/{user_id}", mdAuth(mdClear(MANAGER, cfg.handleRemoveBudgetMember)))
	mux.HandleFunc("GET /api/budgets", mdAuth(cfg.handleGetUserBudgets))
	mux.HandleFunc("GET /api/budgets/{budget_id}", mdAuth(mdClear(VIEWER, cfg.handleGetBudget)))
	mux.HandleFunc("GET /api/budgets/{budget_id}/capital", mdAuth(mdClear(VIEWER, cfg.handleGetBudgetCapital)))
	// Groups & Categories
	mux.HandleFunc("POST /api/budgets/{budget_id}/groups", mdAuth(mdClear(MANAGER, cfg.handleCreateGroup)))
	mux.HandleFunc("POST /api/budgets/{budget_id}/categories", mdAuth(mdClear(MANAGER, cfg.handleCreateCategory)))
	mux.HandleFunc("GET /api/budgets/{budget_id}/groups", mdAuth(mdClear(VIEWER, cfg.handleGetGroups)))
	mux.HandleFunc("GET /api/budgets/{budget_id}/categories", mdAuth(mdClear(VIEWER, cfg.handleGetCategories)))
	mux.HandleFunc("PUT /api/budgets/{budget_id}/groups/{group_id}", mdAuth(mdClear(MANAGER, cfg.handleUpdateGroup)))
	mux.HandleFunc("PUT /api/budgets/{budget_id}/categories/{category_id}", mdAuth(mdClear(MANAGER, cfg.handleUpdateCategory)))
	mux.HandleFunc("DELETE /api/budgets/{budget_id}/groups/{group_id}", mdAuth(mdClear(MANAGER, cfg.handleDeleteGroup)))
	mux.HandleFunc("DELETE /api/budgets/{budget_id}/categories/{category_id}", mdAuth(mdClear(MANAGER, cfg.handleDeleteCategory)))
	// Payees
	mux.HandleFunc("POST /api/budgets/{budget_id}/payees", mdAuth(mdClear(CONTRIBUTOR, cfg.handleCreatePayee)))
	mux.HandleFunc("GET /api/budgets/{budget_id}/payees", mdAuth(mdClear(VIEWER, cfg.handleGetPayees)))
	mux.HandleFunc("GET /api/budgets/{budget_id}/payees/{payee_id}", mdAuth(mdClear(VIEWER, cfg.handleGetPayee)))
	mux.HandleFunc("PUT /api/budgets/{budget_id}/payees/{payee_id}", mdAuth(mdClear(MANAGER, cfg.handleUpdatePayee)))
	mux.HandleFunc("DELETE /api/budgets/{budget_id}/payees/{payee_id}", mdAuth(mdClear(CONTRIBUTOR, cfg.handleDeletePayee)))
	// Accounts
	mux.HandleFunc("POST /api/budgets/{budget_id}/accounts", mdAuth(mdClear(MANAGER, cfg.handleAddAccount)))
	mux.HandleFunc("GET /api/budgets/{budget_id}/accounts", mdAuth(mdClear(VIEWER, cfg.handleGetAccounts)))
	mux.HandleFunc("GET /api/budgets/{budget_id}/accounts/{account_id}", mdAuth(mdClear(VIEWER, cfg.handleGetAccount)))
	mux.HandleFunc("GET /api/budgets/{budget_id}/accounts/{account_id}/capital", mdAuth(mdClear(VIEWER, cfg.handleGetBudgetAccountCapital)))
	mux.HandleFunc("PUT /api/budgets/{budget_id}/accounts/{account_id}", mdAuth(mdClear(MANAGER, cfg.handleUpdateAccount)))
	mux.HandleFunc("PATCH /api/budgets/{budget_id}/accounts/{account_id}", mdAuth(mdClear(MANAGER, cfg.handleRestoreAccount)))
	mux.HandleFunc("DELETE /api/budgets/{budget_id}/accounts/{account_id}", mdAuth(mdClear(MANAGER, cfg.handleDeleteAccount)))
	// Transactions
	mux.HandleFunc("POST /api/budgets/{budget_id}/transactions", mdAuth(mdClear(CONTRIBUTOR, mdValidateTxn(cfg.handleLogTransaction))))
	mux.HandleFunc("GET /api/budgets/{budget_id}/transactions", mdAuth(mdClear(VIEWER, cfg.handleGetTransactions)))
	mux.HandleFunc("GET /api/budgets/{budget_id}/transactions/details", mdAuth(mdClear(VIEWER, cfg.handleGetTransactions)))
	mux.HandleFunc("GET /api/budgets/{budget_id}/transactions/{transaction_id}", mdAuth(mdClear(VIEWER, cfg.handleGetTransaction)))
	mux.HandleFunc("GET /api/budgets/{budget_id}/transactions/{transaction_id}/details", mdAuth(mdClear(VIEWER, cfg.handleGetTransaction)))
	mux.HandleFunc("GET /api/budgets/{budget_id}/transactions/{transaction_id}/splits", mdAuth(mdClear(VIEWER, cfg.handleGetTransactionSplits)))
	mux.HandleFunc("PUT /api/budgets/{budget_id}/transactions/{transaction_id}", mdAuth(mdClear(MANAGER, mdValidateTxn(cfg.handleUpdateTransaction))))
	mux.HandleFunc("DELETE /api/budgets/{budget_id}/transactions/{transaction_id}", mdAuth(mdClear(CONTRIBUTOR, cfg.handleDeleteTransaction)))
	// Months & Dollar Assignment
	mux.HandleFunc("POST /api/budgets/{budget_id}/months/{month_id}/categories", mdAuth(mdClear(MANAGER, cfg.handleAssignAmountToCategory)))
	mux.HandleFunc("GET /api/budgets/{budget_id}/months/{month_id}/categories/{category_id}", mdAuth(mdClear(VIEWER, cfg.handleGetMonthCategoryReport)))
	mux.HandleFunc("GET /api/budgets/{budget_id}/months/{month_id}/categories", mdAuth(mdClear(VIEWER, cfg.handleGetMonthCategories)))
	mux.HandleFunc("GET /api/budgets/{budget_id}/months/{month_id}", mdAuth(mdClear(VIEWER, cfg.handleGetMonthReport)))
	return mux
}
