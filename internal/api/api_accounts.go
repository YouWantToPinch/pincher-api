package api

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/YouWantToPinch/pincher-api/internal/database"
)

func (cfg *apiConfig) endpAddAccount(w http.ResponseWriter, r *http.Request) {

	type parameters struct {
		AccountType string `json:"account_type"`
		Name        string `json:"name"`
		Notes       string `json:"notes"`
	}

	params, err := decodeParams[parameters](r)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failure decoding parameters", err)
		return
	}

	pathBudgetID := getContextKeyValue(r.Context(), "budget_id")

	dbAccount, err := cfg.db.AddAccount(r.Context(), database.AddAccountParams{
		BudgetID:    pathBudgetID,
		AccountType: params.AccountType,
		Name:        params.Name,
		Notes:       params.Notes,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create account", err)
		return
	}

	respBody := Account{
		ID:          dbAccount.ID,
		CreatedAt:   dbAccount.CreatedAt,
		UpdatedAt:   dbAccount.UpdatedAt,
		AccountType: dbAccount.AccountType,
		Name:        dbAccount.Name,
		Notes:       dbAccount.Notes,
		IsDeleted:   dbAccount.IsDeleted,
	}

	respondWithJSON(w, http.StatusCreated, respBody)
}

func (cfg *apiConfig) endpGetAccounts(w http.ResponseWriter, r *http.Request) {

	pathBudgetID := getContextKeyValue(r.Context(), "budget_id")
	dbAccounts, err := cfg.db.GetAccountsFromBudget(r.Context(), pathBudgetID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get budget accounts", err)
		return
	}

	queryInclude := r.URL.Query().Get("include")

	var accounts []Account
	for _, account := range dbAccounts {
		if account.IsDeleted && queryInclude != "deleted" {
			continue
		}
		accounts = append(accounts, Account{
			ID:          account.ID,
			CreatedAt:   account.CreatedAt,
			UpdatedAt:   account.UpdatedAt,
			AccountType: account.AccountType,
			Name:        account.Name,
			Notes:       account.Notes,
			IsDeleted:   account.IsDeleted,
		})
	}

	type resp struct {
		Accounts []Account `json:"accounts"`
	}

	respBody := resp{
		Accounts: accounts,
	}

	respondWithJSON(w, http.StatusOK, respBody)
}

func (cfg *apiConfig) endpGetAccount(w http.ResponseWriter, r *http.Request) {

	idString := r.PathValue("account_id")
	pathAccountID, err := uuid.Parse(idString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid id", err)
		return
	}

	dbAccount, err := cfg.db.GetAccountByID(r.Context(), pathAccountID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Couldn't find account with specified id", err)
		return
	}

	respBody := Account{
		ID:          dbAccount.ID,
		CreatedAt:   dbAccount.CreatedAt,
		UpdatedAt:   dbAccount.UpdatedAt,
		AccountType: dbAccount.AccountType,
		Name:        dbAccount.Name,
		Notes:       dbAccount.Notes,
		IsDeleted:   dbAccount.IsDeleted,
	}

	respondWithJSON(w, http.StatusCreated, respBody)
}

func (cfg *apiConfig) endpGetBudgetAccountCapital(w http.ResponseWriter, r *http.Request) {

	idString := r.PathValue("account_id")
	pathAccountID, err := uuid.Parse(idString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid id", err)
		return
	}

	capitalAmount, err := cfg.db.GetBudgetAccountCapital(r.Context(), pathAccountID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong", err)
		return
	}

	type response struct {
		Capital int64 `json:"capital"`
	}

	respBody := response{
		Capital: capitalAmount,
	}

	respondWithJSON(w, http.StatusOK, respBody)
}

func (cfg *apiConfig) endpUpdateAccount(w http.ResponseWriter, r *http.Request) {
	var pathAccountID uuid.UUID
	err := parseUUIDFromPath("account_id", r, &pathAccountID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid id", err)
		return
	}

	type parameters struct {
		AccountType string `json:"account_type"`
		Name        string `json:"name"`
		Notes       string `json:"notes"`
	}

	params, err := decodeParams[parameters](r)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failure decoding parameters", err)
		return
	}

	_, err = cfg.db.UpdateAccount(r.Context(), database.UpdateAccountParams{
		ID:          pathAccountID,
		AccountType: params.AccountType,
		Name:        params.Name,
		Notes:       params.Notes,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update account", err)
		return
	}

	respondWithText(w, http.StatusNoContent, "Account '"+params.Name+"' updated successfully!")
}

func (cfg *apiConfig) endpDeleteAccount(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Name       string `json:"name"`
		DeleteHard bool   `json:"delete_hard"`
	}

	params, err := decodeParams[parameters](r)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failure decoding parameters", err)
		return
	}

	idString := r.PathValue("account_id")
	pathAccountID, err := uuid.Parse(idString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid id", err)
		return
	}

	dbAccount, err := cfg.db.GetAccountByID(r.Context(), pathAccountID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Couldn't find account with specified id", err)
		return
	}

	pathBudgetID := getContextKeyValue(r.Context(), "budget_id")
	if pathBudgetID != dbAccount.BudgetID {
		respondWithError(w, http.StatusForbidden, "401 Unauthorized", nil)
		return
	}

	if !params.DeleteHard {
		err = cfg.db.DeleteAccountSoft(r.Context(), pathAccountID)
		if err != nil {
			respondWithError(w, http.StatusNotFound, "Account with id specified not found", nil)
			return
		}
		respondWithText(w, http.StatusNoContent, "The account was deleted. It may be restored.")
		return
	} else {
		if !dbAccount.IsDeleted {
			respondWithError(w, http.StatusBadRequest, "Request for hard delete ignored; a soft delete is required first", nil)
			return
		}
		if params.Name != dbAccount.Name {
			respondWithError(w, http.StatusBadRequest, "Request for hard delete ignored; input name does not match name within database", nil)
			return
		}

		err = cfg.db.DeleteAccountHard(r.Context(), pathAccountID)
		if err != nil {
			respondWithError(w, http.StatusNotFound, "404 Not Found", err)
			return
		}
		respondWithText(w, http.StatusNoContent, "The account was deleted and cannot be restored.")
		return
	}
}
