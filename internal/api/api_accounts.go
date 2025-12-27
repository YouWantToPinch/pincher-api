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
		respondWithError(w, http.StatusInternalServerError, "failure decoding request payload: ", err)
		return
	}

	if rqPayload.Name == "" {
		respondWithError(w, http.StatusBadRequest, "name not provided", nil)
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
		respondWithError(w, http.StatusInternalServerError, "could not create account: ", err)
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
		respondWithError(w, http.StatusInternalServerError, "could not get budget accounts: ", err)
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
		respondWithError(w, http.StatusBadRequest, "failure parsing account_id as UUID: ", err)
		return
	}

	dbAccount, err := cfg.db.GetAccountByID(r.Context(), pathAccountID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "could not find account: ", err)
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
		respondWithError(w, http.StatusBadRequest, "failure parsing account_id as UUID: ", err)
		return
	}

	capitalAmount, err := cfg.db.GetBudgetAccountCapital(r.Context(), pathAccountID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not get budget account capital: ", err)
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
		respondWithError(w, http.StatusBadRequest, "failure parsing UUID: ", err)
		return
	}

	type rqSchema struct {
		AccountType string `json:"account_type"`
		Meta
	}

	rqPayload, err := decodePayload[rqSchema](r)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failure decoding request payload: ", err)
		return
	}

	_, err = cfg.db.UpdateAccount(r.Context(), database.UpdateAccountParams{
		ID:          pathAccountID,
		AccountType: rqPayload.AccountType,
		Name:        rqPayload.Name,
		Notes:       rqPayload.Notes,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not update account: ", err)
		return
	}

	respondWithText(w, http.StatusOK, "Account '"+rqPayload.Name+"' updated successfully")
}

func (cfg *APIConfig) endpRestoreAccount(w http.ResponseWriter, r *http.Request) {
	var pathAccountID uuid.UUID
	err := parseUUIDFromPath("account_id", r, &pathAccountID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "failure parsing UUID: ", err)
		return
	}

	err = cfg.db.RestoreAccount(r.Context(), pathAccountID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not restore account: ", err)
		return
	}

	respondWithText(w, http.StatusOK, "Account restored")
}

func (cfg *APIConfig) endpDeleteAccount(w http.ResponseWriter, r *http.Request) {
	type rqSchema struct {
		Name       string `json:"name"`
		DeleteHard bool   `json:"delete_hard"`
	}

	rqPayload, err := decodePayload[rqSchema](r)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failure decoding request payload: ", err)
		return
	}

	idString := r.PathValue("account_id")
	pathAccountID, err := uuid.Parse(idString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "failure parsing UUID: ", err)
		return
	}

	dbAccount, err := cfg.db.GetAccountByID(r.Context(), pathAccountID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "could not find account: ", err)
		return
	}

	pathBudgetID := getContextKeyValue(r.Context(), "budget_id")
	if pathBudgetID != dbAccount.BudgetID {
		respondWithError(w, http.StatusForbidden, "401 Unauthorized", nil)
		return
	}

	if !rqPayload.DeleteHard {
		if dbAccount.IsDeleted {
			respondWithText(w, http.StatusOK, "Account already deleted.")
			return
		}
		err = cfg.db.DeleteAccountSoft(r.Context(), pathAccountID)
		if err != nil {
			respondWithError(w, http.StatusNotFound, "could not find account: ", nil)
			return
		}
		respondWithText(w, http.StatusOK, "Account soft-deleted; it may be restored")
		return
	} else {
		if !dbAccount.IsDeleted {
			respondWithError(w, http.StatusBadRequest, "request for hard delete ignored (soft is required first)", nil)
			return
		}
		if rqPayload.Name != dbAccount.Name {
			respondWithError(w, http.StatusBadRequest, "request for hard delete ignored (input name must match name within database)", nil)
			return
		}

		err = cfg.db.DeleteAccountHard(r.Context(), pathAccountID)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "could not delete account: ", err)
			return
		}
		respondWithText(w, http.StatusOK, "Account hard-deleted successfully; it cannot be restored")
		return
	}
}
