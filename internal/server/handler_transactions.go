package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/YouWantToPinch/pincher-api/internal/database"
)

func (cfg *apiConfig) endpLogTransaction(w http.ResponseWriter, r *http.Request) {

	//body, _ := io.ReadAll(r.Body)
	//fmt.Println("Raw request body:", string(body))

	type parameters struct {
		AccountID       string    `json:"account_id"`
		TransactionDate time.Time `json:"transaction_date"`
		PayeeID         string    `json:"payee_id"`
		Notes           string    `json:"notes"`
		Cleared         string    `json:"is_cleared"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failure decoding parameters when logging transaction", err)
		return
	}

	parsedAccountID, err := uuid.Parse(params.AccountID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Provided account_id string could not be parsed as UUID", err)
		return
	}
	parsedPayeeID, err := uuid.Parse(params.PayeeID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Provided payee_id string could not be parsed as UUID", err)
		return
	}
	var parsedCleared bool
	if params.Cleared == "true" || params.Cleared == "false" {
		parsedCleared, err = strconv.ParseBool(params.Cleared)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Provided string value for 'Cleared' could not be parsed as boolean", err)
		}
	} else {
		respondWithError(w, http.StatusBadRequest, "Provided string value for 'Cleared' not 'true' or 'false'; cannot be parsed", nil)
		return
	}

	validatedUserID := getContextKeyValue(r.Context(), "user_id")
	pathBudgetID := getContextKeyValue(r.Context(), "budget_id")

	dbTransaction, err := cfg.db.LogTransaction(r.Context(), database.LogTransactionParams{
		BudgetID:        pathBudgetID,
		LoggerID:        validatedUserID,
		AccountID:       parsedAccountID,
		TransactionDate: params.TransactionDate,
		PayeeID:         parsedPayeeID,
		Notes:           params.Notes,
		Cleared:         parsedCleared,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't log transaction", err)
		return
	}

	respBody := Transaction{
		ID:              dbTransaction.ID,
		CreatedAt:       dbTransaction.CreatedAt,
		UpdatedAt:       dbTransaction.UpdatedAt,
		BudgetID:        dbTransaction.BudgetID,
		LoggerID:        dbTransaction.LoggerID,
		AccountID:       dbTransaction.AccountID,
		TransactionDate: dbTransaction.TransactionDate,
		PayeeID:         dbTransaction.PayeeID,
		Notes:           dbTransaction.Notes,
		Cleared:         dbTransaction.Cleared,
	}

	respondWithJSON(w, http.StatusCreated, respBody)
	return
}

func (cfg *apiConfig) endpGetTransactions(w http.ResponseWriter, r *http.Request) {

	var err error
	queryStartDate := r.URL.Query().Get("start_date")
	var parsedStartDate time.Time
	queryEndDate := r.URL.Query().Get("end_date")
	var parsedEndDate time.Time
	queryAccountID := r.URL.Query().Get("account_id")
	var parsedAccountID uuid.UUID
	//queryCategoryID := r.URL.Query().Get("category_id")

	if queryAccountID != "" {
		parsedAccountID, err = uuid.Parse(queryAccountID)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Provided account_id string could not be parsed as UUID", err)
			return
		}
	} else {
		parsedAccountID = uuid.Nil
	}
	if queryStartDate != "" || queryEndDate != "" {
		const timeLayout = time.RFC3339
		parsedStartDate, err = time.Parse(timeLayout, queryStartDate)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Could not parse Start Date provided", err)
			return
		}
		parsedEndDate, err = time.Parse(timeLayout, queryEndDate)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Could not parse End Date provided", err)
			return
		}
	} else {
		parsedStartDate = time.Time{}
		parsedEndDate = time.Time{}
	}
	pathBudgetID := getContextKeyValue(r.Context(), "budget_id")
	slog.Debug("pathBudgetID: " + pathBudgetID.String())

	slog.Debug(fmt.Sprintf("Transaction paramaters: budget_id=%s, account_id=%v (nil=%v), start_date=%v (zero=%v), end_date=%v (zero=%v)",
		pathBudgetID.String(),
		parsedAccountID,
		parsedAccountID == uuid.Nil,
		parsedStartDate,
		parsedStartDate.IsZero(),
		parsedEndDate,
		parsedEndDate.IsZero()),
	)

	transactions, err := cfg.db.GetTransactions(r.Context(), database.GetTransactionsParams{
		AccountID: parsedAccountID,
		StartDate: parsedStartDate,
		EndDate:   parsedEndDate,
		BudgetID:  pathBudgetID,
	})
	if err != nil {
		respondWithError(w, http.StatusNotFound, "No transactions found", err)
		return
	}

	var respBody []Transaction
	for _, transaction := range transactions {
		addTransaction := Transaction{
			ID:              transaction.ID,
			CreatedAt:       transaction.CreatedAt,
			UpdatedAt:       transaction.UpdatedAt,
			BudgetID:        transaction.BudgetID,
			LoggerID:        transaction.LoggerID,
			AccountID:       transaction.AccountID,
			TransactionDate: transaction.TransactionDate,
			PayeeID:         transaction.PayeeID,
			Notes:           transaction.Notes,
			Cleared:         transaction.Cleared,
		}
		respBody = append(respBody, addTransaction)
	}

	slog.Debug(fmt.Sprintf("TRANSACTIONS FOUND: %d", len(respBody)))

	respondWithJSON(w, http.StatusOK, respBody)
	return
}

func (cfg *apiConfig) endpGetTransaction(w http.ResponseWriter, r *http.Request) {
	idString := r.PathValue("transaction_id")
	pathTransactionID, err := uuid.Parse(idString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid id", err)
		return
	}

	dbTransaction, err := cfg.db.GetTransactionByID(r.Context(), pathTransactionID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Couldn't find transaction with specified id", err)
		return
	}

	respBody := Transaction{
		ID:              dbTransaction.ID,
		CreatedAt:       dbTransaction.CreatedAt,
		UpdatedAt:       dbTransaction.UpdatedAt,
		BudgetID:        dbTransaction.BudgetID,
		LoggerID:        dbTransaction.LoggerID,
		AccountID:       dbTransaction.AccountID,
		TransactionDate: dbTransaction.TransactionDate,
		PayeeID:         dbTransaction.PayeeID,
		Notes:           dbTransaction.Notes,
		Cleared:         dbTransaction.Cleared,
	}

	respondWithJSON(w, http.StatusCreated, respBody)
	return
}

func (cfg *apiConfig) endpDeleteTransaction(w http.ResponseWriter, r *http.Request) {
	idString := r.PathValue("transaction_id")
	pathTransactionID, err := uuid.Parse(idString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid id", err)
		return
	}

	dbTransaction, err := cfg.db.GetTransactionByID(r.Context(), pathTransactionID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Couldn't find transaction with specified id", err)
		return
	}
	pathBudgetID := getContextKeyValue(r.Context(), "budget_id")
	if pathBudgetID != dbTransaction.BudgetID {
		respondWithError(w, http.StatusForbidden, "401 Unauthorized", nil)
		return
	}

	err = cfg.db.DeleteTransaction(r.Context(), pathTransactionID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "404 Not Found", err)
		return
	}
	respondWithText(w, http.StatusNoContent, "The transaction was deleted")
	return
}
