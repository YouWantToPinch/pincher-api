// Package api handles routes and their associated handlers
package api

import (
	"net/http"

	reg "github.com/YouWantToPinch/pincher-api/internal/registrar"
)

func SetupMux(cfg *APIConfig) http.Handler {
	mux := http.NewServeMux()

	// middleware
	mdAuth := cfg.middlewareAuthenticate
	mdClear := cfg.middlewareCheckClearance
	mdValidateTxn := cfg.middlewareValidateTxn

	// REGISTER API HANDLERS
	// ======================

	r, err := reg.NewRegistrar(mux)
	if err != nil {
		panic(err)
	}

	admin, err := reg.NewBuilder("admin")
	if err != nil {
		panic(err)
	}
	api, err := reg.NewBuilder("api")
	if err != nil {
		panic(err)
	}

	// Admin & State
	r.Handle(
		admin.Build().Post().Add("reset"),
		cfg.handleDeleteAllUsers,
	)
	r.Handle(
		admin.Build().Get().Add("users"),
		cfg.handleGetAllUsers,
	)
	r.Handle(
		admin.Build().Get().Add("users").Add("count"),
		cfg.handleGetTotalUserCount,
	)
	r.Handle(
		api.Build().Get().Add("healthz"),
		cfg.handleReadiness,
	)

	// User authentication
	r.Handle(
		api.Build().Post().Add("users"),
		cfg.handleCreateUser,
	)
	r.Handle(
		api.Build().Delete().Add("users"),
		mdAuth(cfg.handleDeleteUser),
	)
	r.Handle(
		api.Build().Put().Add("users"),
		mdAuth(cfg.handleUpdateUserCredentials),
	)
	r.Handle(
		api.Build().Post().Add("login"),
		cfg.handleLoginUser,
	)
	r.Handle(
		api.Build().Post().Add("refresh"),
		cfg.handleCheckRefreshToken,
	)
	r.Handle(
		api.Build().Post().Add("revoke"),
		cfg.handleRevokeRefreshToken,
	)
	// Budgets
	r.Handle(
		api.Build().Post().Budget().Col(),
		mdAuth(cfg.handleCreateBudget),
	)
	r.Handle(
		api.Build().Post().Budget().Member().Col(),
		mdAuth(mdClear(MANAGER, cfg.handleAddBudgetMemberWithRole)),
	)
	r.Handle(
		api.Build().Put().Budget(),
		mdAuth(mdClear(MANAGER, cfg.handleUpdateBudget)),
	)
	r.Handle(
		api.Build().Delete().Budget(),
		mdAuth(mdClear(ADMIN, cfg.handleDeleteBudget)),
	)
	r.Handle(
		api.Build().Delete().Budget().Member(),
		mdAuth(mdClear(MANAGER, cfg.handleRemoveBudgetMember)),
	)
	r.Handle(
		api.Build().Get().Budget().Col(),
		mdAuth(cfg.handleGetUserBudgets),
	)
	r.Handle(
		api.Build().Get().Budget(),
		mdAuth(mdClear(VIEWER, cfg.handleGetBudget)),
	)
	r.Handle(
		api.Build().Get().Budget().Add("capital"),
		mdAuth(mdClear(VIEWER, cfg.handleGetBudgetCapital)),
	)
	// Groups
	r.Handle(
		api.Build().Post().Budget().Group().Col(),
		mdAuth(mdClear(MANAGER, cfg.handleCreateGroup)),
	)
	r.Handle(
		api.Build().Get().Budget().Group().Col(),
		mdAuth(mdClear(VIEWER, cfg.handleGetGroups)),
	)
	r.Handle(
		api.Build().Get().Budget().Group(),
		mdAuth(mdClear(VIEWER, cfg.handleGetGroup)),
	)
	r.Handle(
		api.Build().Put().Budget().Group(),
		mdAuth(mdClear(MANAGER, cfg.handleUpdateGroup)),
	)
	r.Handle(
		api.Build().Delete().Budget().Group(),
		mdAuth(mdClear(MANAGER, cfg.handleDeleteGroup)),
	)
	// Categories
	r.Handle(
		api.Build().Post().Budget().Category().Col(),
		mdAuth(mdClear(MANAGER, cfg.handleCreateCategory)),
	)
	r.Handle(
		api.Build().Get().Budget().Category().Col(),
		mdAuth(mdClear(VIEWER, cfg.handleGetCategories)),
	)
	r.Handle(
		api.Build().Get().Budget().Category(),
		mdAuth(mdClear(VIEWER, cfg.handleGetCategory)),
	)
	r.Handle(
		api.Build().Put().Budget().Category(),
		mdAuth(mdClear(MANAGER, cfg.handleUpdateCategory)),
	)
	r.Handle(
		api.Build().Delete().Budget().Category(),
		mdAuth(mdClear(MANAGER, cfg.handleDeleteCategory)),
	)
	// Payees
	r.Handle(
		api.Build().Post().Budget().Payee().Col(),
		mdAuth(mdClear(CONTRIBUTOR, cfg.handleCreatePayee)),
	)
	r.Handle(
		api.Build().Get().Budget().Payee().Col(),
		mdAuth(mdClear(VIEWER, cfg.handleGetPayees)),
	)
	r.Handle(
		api.Build().Get().Budget().Payee(),
		mdAuth(mdClear(VIEWER, cfg.handleGetPayee)),
	)
	r.Handle(
		api.Build().Put().Budget().Payee(),
		mdAuth(mdClear(MANAGER, cfg.handleUpdatePayee)),
	)
	r.Handle(
		api.Build().Delete().Budget().Payee(),
		mdAuth(mdClear(CONTRIBUTOR, cfg.handleDeletePayee)),
	)
	// Accounts
	r.Handle(
		api.Build().Post().Budget().Account().Col(),
		mdAuth(mdClear(CONTRIBUTOR, cfg.handleAddAccount)),
	)
	r.Handle(
		api.Build().Get().Budget().Account().Col(),
		mdAuth(mdClear(VIEWER, cfg.handleGetAccounts)),
	)
	r.Handle(
		api.Build().Get().Budget().Account(),
		mdAuth(mdClear(VIEWER, cfg.handleGetAccount)),
	)
	r.Handle(
		api.Build().Get().Budget().Account().Add("capital"),
		mdAuth(mdClear(VIEWER, cfg.handleGetBudgetAccountCapital)),
	)
	r.Handle(
		api.Build().Put().Budget().Account(),
		mdAuth(mdClear(MANAGER, cfg.handleUpdateAccount)),
	)
	r.Handle(
		api.Build().Patch().Budget().Account(),
		mdAuth(mdClear(MANAGER, cfg.handleRestoreAccount)),
	)
	r.Handle(
		api.Build().Delete().Budget().Account(),
		mdAuth(mdClear(MANAGER, cfg.handleDeleteAccount)),
	)
	// Transactions
	r.Handle(
		api.Build().Post().Budget().Transaction().Col(),
		mdAuth(mdClear(CONTRIBUTOR, mdValidateTxn(cfg.handleLogTransaction))),
	)
	r.Handle(
		api.Build().Get().Budget().Transaction().Col(),
		mdAuth(mdClear(VIEWER, cfg.handleGetTransactions)),
	)
	r.Handle(
		api.Build().Get().Budget().Transaction().Col().Add("details"),
		mdAuth(mdClear(VIEWER, cfg.handleGetTransactions)),
	)
	r.Handle(
		api.Build().Get().Budget().Transaction(),
		mdAuth(mdClear(VIEWER, cfg.handleGetTransaction)),
	)
	r.Handle(
		api.Build().Get().Budget().Transaction().Add("details"),
		mdAuth(mdClear(VIEWER, cfg.handleGetTransaction)),
	)
	r.Handle(
		api.Build().Get().Budget().Transaction().Add("splits"),
		mdAuth(mdClear(VIEWER, cfg.handleGetTransactionSplits)),
	)
	r.Handle(
		api.Build().Put().Budget().Transaction(),
		mdAuth(mdClear(MANAGER, mdValidateTxn(cfg.handleUpdateTransaction))),
	)
	r.Handle(
		api.Build().Delete().Budget().Transaction(),
		mdAuth(mdClear(MANAGER, cfg.handleDeleteTransaction)),
	)

	// Dollar Assignment
	r.Handle(
		api.Build().Post().Budget().Month().Category().Col(),
		mdAuth(mdClear(MANAGER, cfg.handleAssignAmountToCategory)),
	)
	// Reporting
	r.Handle(
		api.Build().Get().Budget().Month().Category(),
		mdAuth(mdClear(VIEWER, cfg.handleGetMonthCategoryReport)),
	)
	r.Handle(
		api.Build().Get().Budget().Month().Category().Col(),
		mdAuth(mdClear(VIEWER, cfg.handleGetMonthCategories)),
	)
	r.Handle(
		api.Build().Get().Budget().Month().Group(),
		mdAuth(mdClear(VIEWER, cfg.handleGetMonthGroupReport)),
	)
	r.Handle(
		api.Build().Get().Budget().Month().Group().Col(),
		mdAuth(mdClear(VIEWER, cfg.handleGetMonthGroups)),
	)
	r.Handle(
		api.Build().Get().Budget().Month(),
		mdAuth(mdClear(VIEWER, cfg.handleGetMonthReport)),
	)

	handler := cfg.middlewareHandleCORS(mux)

	return handler
}
