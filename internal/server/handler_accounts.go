package server

import (
	//"log"
	"database/sql"
	"encoding/json"
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

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failure decoding parameters", err)
		return
	}

	pathBudgetID := getContextKeyValue(r.Context(), "budget_id")

	dbAccount, err := cfg.db.AddAccount(r.Context(), database.AddAccountParams{
		BudgetID:    pathBudgetID,
		AccountType: params.AccountType,
		Name:        params.Name,
		Notes: sql.NullString{
			String: params.Notes,
			Valid:  true,
		},
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
		Notes:       dbAccount.Notes.String,
		IsDeleted:   dbAccount.IsDeleted,
	}

	respondWithJSON(w, http.StatusCreated, respBody)
	return
}

func (cfg *apiConfig) endpGetAccounts(w http.ResponseWriter, r *http.Request) {

	pathBudgetID := getContextKeyValue(r.Context(), "budget_id")
	accounts, err := cfg.db.GetAccountsFromBudget(r.Context(), pathBudgetID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get budget accounts", err)
		return
	}

	queryInclude := r.URL.Query().Get("include")

	var respBody []Account
	for _, account := range accounts {
		if account.IsDeleted && queryInclude != "deleted" {
			continue
		}
		addAccount := Account{
			ID:          account.ID,
			CreatedAt:   account.CreatedAt,
			UpdatedAt:   account.UpdatedAt,
			AccountType: account.AccountType,
			Name:        account.Name,
			Notes:       account.Notes.String,
			IsDeleted:   account.IsDeleted,
		}
		respBody = append(respBody, addAccount)
	}

	respondWithJSON(w, http.StatusOK, respBody)
	return
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
		Notes:       dbAccount.Notes.String,
		IsDeleted:   dbAccount.IsDeleted,
	}

	respondWithJSON(w, http.StatusCreated, respBody)
	return
}

func (cfg *apiConfig) endpDeleteAccount(w http.ResponseWriter, r *http.Request) {

	type parameters struct {
		Name       string `json:"name"`
		DeleteHard bool   `json:"delete_hard"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
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
