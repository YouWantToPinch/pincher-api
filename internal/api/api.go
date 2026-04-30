// Package api handles routes and their associated handlers
package api

import (
	"fmt"
	"log/slog"
	"net/http"
)

type registrar struct {
	registry map[string]struct{} // patterns already registered as endpoints
	mux      *http.ServeMux      // request multiplexer to register inputs to
}

func (r *registrar) register(pattern string, fn http.HandlerFunc) {
	if _, ok := r.registry[pattern]; ok {
		panic(fmt.Sprintf("handler already registered with pattern: %s", pattern))
	}
	r.registry[pattern] = struct{}{}
	r.mux.HandleFunc(pattern, fn)
	slog.Info("Registered handler in mux with pattern: " + pattern)
}

func (r *registrar) validate(ef *endpointFormatter) string {
	if ef == nil || ef.current == "" {
		panic("could not register API handler; nil formatter provided")
	} else if ef.current == "" {
		panic("could not register API handler; no pattern provided")
	}
	if r.mux == nil {
		panic("could not register API handler; mux was nil")
	}
	pattern := ef.end()
	return pattern
}

func (r *registrar) handle(ef *endpointFormatter, fn http.HandlerFunc) {
	r.register(r.validate(ef), fn)
}

func SetupMux(cfg *APIConfig) http.Handler {
	mux := http.NewServeMux()

	// middleware
	mdAuth := cfg.middlewareAuthenticate
	mdClear := cfg.middlewareCheckClearance
	mdValidateTxn := cfg.middlewareValidateTxn

	// REGISTER API HANDLERS
	// ======================

	r := &registrar{
		registry: map[string]struct{}{},
		mux:      mux,
	}
	admin := &endpointFormatter{basePath: "admin"}
	api := &endpointFormatter{basePath: "api"}

	// Admin & State
	r.handle(
		admin.post().add("reset"),
		cfg.handleDeleteAllUsers,
	)
	r.handle(
		admin.get().add("users"),
		cfg.handleGetAllUsers,
	)
	r.handle(
		admin.get().add("users").add("count"),
		cfg.handleGetTotalUserCount,
	)
	r.handle(
		api.get().add("healthz"),
		cfg.handleReadiness,
	)

	// User authentication
	r.handle(
		api.post().add("users"),
		cfg.handleCreateUser,
	)
	r.handle(
		api.delete().add("users"),
		mdAuth(cfg.handleDeleteUser),
	)
	r.handle(
		api.put().add("users"),
		mdAuth(cfg.handleUpdateUserCredentials),
	)
	r.handle(
		api.post().add("login"),
		cfg.handleLoginUser,
	)
	r.handle(
		api.post().add("refresh"),
		cfg.handleCheckRefreshToken,
	)
	r.handle(
		api.post().add("revoke"),
		cfg.handleRevokeRefreshToken,
	)
	// Budgets
	r.handle(
		api.post().budget().col(),
		mdAuth(cfg.handleCreateBudget),
	)
	r.handle(
		api.post().budget().member().col(),
		mdAuth(mdClear(MANAGER, cfg.handleAddBudgetMemberWithRole)),
	)
	r.handle(
		api.put().budget(),
		mdAuth(mdClear(MANAGER, cfg.handleUpdateBudget)),
	)
	r.handle(
		api.delete().budget(),
		mdAuth(mdClear(ADMIN, cfg.handleDeleteBudget)),
	)
	r.handle(
		api.delete().budget().member(),
		mdAuth(mdClear(MANAGER, cfg.handleRemoveBudgetMember)),
	)
	r.handle(
		api.get().budget().col(),
		mdAuth(cfg.handleGetUserBudgets),
	)
	r.handle(
		api.get().budget(),
		mdAuth(mdClear(VIEWER, cfg.handleGetBudget)),
	)
	r.handle(
		api.get().budget().add("capital"),
		mdAuth(mdClear(VIEWER, cfg.handleGetBudgetCapital)),
	)
	// Groups
	r.handle(
		api.post().budget().group().col(),
		mdAuth(mdClear(MANAGER, cfg.handleCreateGroup)),
	)
	r.handle(
		api.get().budget().group().col(),
		mdAuth(mdClear(VIEWER, cfg.handleGetGroups)),
	)
	r.handle(
		api.get().budget().group(),
		mdAuth(mdClear(VIEWER, cfg.handleGetGroup)),
	)
	r.handle(
		api.put().budget().group(),
		mdAuth(mdClear(MANAGER, cfg.handleUpdateGroup)),
	)
	r.handle(
		api.delete().budget().group(),
		mdAuth(mdClear(MANAGER, cfg.handleDeleteGroup)),
	)
	// Categories
	r.handle(
		api.post().budget().category().col(),
		mdAuth(mdClear(MANAGER, cfg.handleCreateCategory)),
	)
	r.handle(
		api.get().budget().category().col(),
		mdAuth(mdClear(VIEWER, cfg.handleGetCategories)),
	)
	r.handle(
		api.get().budget().category(),
		mdAuth(mdClear(VIEWER, cfg.handleGetCategory)),
	)
	r.handle(
		api.put().budget().category(),
		mdAuth(mdClear(MANAGER, cfg.handleUpdateCategory)),
	)
	r.handle(
		api.delete().budget().category(),
		mdAuth(mdClear(MANAGER, cfg.handleDeleteCategory)),
	)
	// Payees
	r.handle(
		api.post().budget().payee().col(),
		mdAuth(mdClear(CONTRIBUTOR, cfg.handleCreatePayee)),
	)
	r.handle(
		api.get().budget().payee().col(),
		mdAuth(mdClear(VIEWER, cfg.handleGetPayees)),
	)
	r.handle(
		api.get().budget().payee(),
		mdAuth(mdClear(VIEWER, cfg.handleGetPayee)),
	)
	r.handle(
		api.put().budget().payee(),
		mdAuth(mdClear(MANAGER, cfg.handleUpdatePayee)),
	)
	r.handle(
		api.delete().budget().payee(),
		mdAuth(mdClear(CONTRIBUTOR, cfg.handleDeletePayee)),
	)
	// Accounts
	r.handle(
		api.post().budget().account().col(),
		mdAuth(mdClear(CONTRIBUTOR, cfg.handleAddAccount)),
	)
	r.handle(
		api.get().budget().account().col(),
		mdAuth(mdClear(VIEWER, cfg.handleGetAccounts)),
	)
	r.handle(
		api.get().budget().account(),
		mdAuth(mdClear(VIEWER, cfg.handleGetAccount)),
	)
	r.handle(
		api.get().budget().account().add("capital"),
		mdAuth(mdClear(VIEWER, cfg.handleGetBudgetAccountCapital)),
	)
	r.handle(
		api.put().budget().account(),
		mdAuth(mdClear(MANAGER, cfg.handleUpdateAccount)),
	)
	r.handle(
		api.patch().budget().account(),
		mdAuth(mdClear(MANAGER, cfg.handleRestoreAccount)),
	)
	r.handle(
		api.delete().budget().account(),
		mdAuth(mdClear(MANAGER, cfg.handleDeleteAccount)),
	)
	// Transactions
	r.handle(
		api.post().budget().transaction().col(),
		mdAuth(mdClear(CONTRIBUTOR, mdValidateTxn(cfg.handleLogTransaction))),
	)
	r.handle(
		api.get().budget().transaction().col(),
		mdAuth(mdClear(VIEWER, cfg.handleGetTransactions)),
	)
	r.handle(
		api.get().budget().transaction().col().add("details"),
		mdAuth(mdClear(VIEWER, cfg.handleGetTransactions)),
	)
	r.handle(
		api.get().budget().transaction(),
		mdAuth(mdClear(VIEWER, cfg.handleGetTransaction)),
	)
	r.handle(
		api.get().budget().transaction().add("details"),
		mdAuth(mdClear(VIEWER, cfg.handleGetTransaction)),
	)
	r.handle(
		api.get().budget().transaction().add("splits"),
		mdAuth(mdClear(VIEWER, cfg.handleGetTransactionSplits)),
	)
	r.handle(
		api.put().budget().transaction(),
		mdAuth(mdClear(MANAGER, mdValidateTxn(cfg.handleUpdateTransaction))),
	)
	r.handle(
		api.delete().budget().transaction(),
		mdAuth(mdClear(MANAGER, cfg.handleDeleteTransaction)),
	)

	// Dollar Assignment
	r.handle(
		api.post().budget().month().category().col(),
		mdAuth(mdClear(MANAGER, cfg.handleAssignAmountToCategory)),
	)
	// Reporting
	r.handle(
		api.get().budget().month().category(),
		mdAuth(mdClear(VIEWER, cfg.handleGetMonthCategoryReport)),
	)
	r.handle(
		api.get().budget().month().category().col(),
		mdAuth(mdClear(VIEWER, cfg.handleGetMonthCategories)),
	)
	r.handle(
		api.get().budget().month().group(),
		mdAuth(mdClear(VIEWER, cfg.handleGetMonthGroupReport)),
	)
	r.handle(
		api.get().budget().month().group().col(),
		mdAuth(mdClear(VIEWER, cfg.handleGetMonthGroups)),
	)
	r.handle(
		api.get().budget().month(),
		mdAuth(mdClear(VIEWER, cfg.handleGetMonthReport)),
	)

	handler := cfg.middlewareHandleCORS(mux)

	return handler
}
