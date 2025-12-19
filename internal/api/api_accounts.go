package api

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/YouWantToPinch/pincher-api/internal/database"
)

func (cfg *APIConfig) endpAddAccount(w http.ResponseWriter, r *http.Request) {
	type rqSchema struct {
		AccountType string `json:"account_type"`
		Meta
	}

	rqPayload, err := decodePayload[rqSchema](r)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failure decoding parameters", err)
		return
	}

	if rqPayload.Name == "" {
		respondWithError(w, http.StatusBadRequest, "Name not provided", nil)
		return
	}

	pathBudgetID := getContextKeyValue(r.Context(), "budget_id")

	dbAccount, err := cfg.db.AddAccount(r.Context(), database.AddAccountParams{
		BudgetID:    pathBudgetID,
		AccountType: rqPayload.AccountType,
		Name:        rqPayload.Name,
		Notes:       rqPayload.Notes,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create account", err)
		return
	}

	rspPayload := Account{
		ID:          dbAccount.ID,
		CreatedAt:   dbAccount.CreatedAt,
		UpdatedAt:   dbAccount.UpdatedAt,
		AccountType: dbAccount.AccountType,
		IsDeleted:   dbAccount.IsDeleted,
		Meta: Meta{
			Name:  dbAccount.Name,
			Notes: dbAccount.Notes,
		},
	}

	respondWithJSON(w, http.StatusCreated, rspPayload)
}

func (cfg *APIConfig) endpGetAccounts(w http.ResponseWriter, r *http.Request) {
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
			IsDeleted:   account.IsDeleted,
			Meta: Meta{
				Name:  account.Name,
				Notes: account.Notes,
			},
		})
	}

	type rspSchema struct {
		Accounts []Account `json:"accounts"`
	}

	rspPayload := rspSchema{
		Accounts: accounts,
	}

	respondWithJSON(w, http.StatusOK, rspPayload)
}

func (cfg *APIConfig) endpGetAccount(w http.ResponseWriter, r *http.Request) {
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

	rspPayload := Account{
		ID:          dbAccount.ID,
		CreatedAt:   dbAccount.CreatedAt,
		UpdatedAt:   dbAccount.UpdatedAt,
		AccountType: dbAccount.AccountType,
		IsDeleted:   dbAccount.IsDeleted,
		Meta: Meta{
			Name:  dbAccount.Name,
			Notes: dbAccount.Notes,
		},
	}

	respondWithJSON(w, http.StatusCreated, rspPayload)
}

func (cfg *APIConfig) endpGetBudgetAccountCapital(w http.ResponseWriter, r *http.Request) {
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

	type rspSchema struct {
		Capital int64 `json:"capital"`
	}

	rspPayload := rspSchema{
		Capital: capitalAmount,
	}

	respondWithJSON(w, http.StatusOK, rspPayload)
}

func (cfg *APIConfig) endpUpdateAccount(w http.ResponseWriter, r *http.Request) {
	var pathAccountID uuid.UUID
	err := parseUUIDFromPath("account_id", r, &pathAccountID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid id", err)
		return
	}

	type rqSchema struct {
		AccountType string `json:"account_type"`
		Meta
	}

	rqPayload, err := decodePayload[rqSchema](r)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failure decoding parameters", err)
		return
	}

	_, err = cfg.db.UpdateAccount(r.Context(), database.UpdateAccountParams{
		ID:          pathAccountID,
		AccountType: rqPayload.AccountType,
		Name:        rqPayload.Name,
		Notes:       rqPayload.Notes,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update account", err)
		return
	}

	respondWithText(w, http.StatusNoContent, "Account '"+rqPayload.Name+"' updated successfully!")
}

func (cfg *APIConfig) endpDeleteAccount(w http.ResponseWriter, r *http.Request) {
	type rqSchema struct {
		Name       string `json:"name"`
		DeleteHard bool   `json:"delete_hard"`
	}

	rqPayload, err := decodePayload[rqSchema](r)
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

	if !rqPayload.DeleteHard {
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
		if rqPayload.Name != dbAccount.Name {
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
