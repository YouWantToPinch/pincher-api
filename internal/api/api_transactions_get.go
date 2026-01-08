package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/YouWantToPinch/pincher-api/internal/database"
)

func (cfg *APIConfig) handleGetTransactionSplits(w http.ResponseWriter, r *http.Request) {
	pathTransactionID, err := parseUUIDFromPath("transaction_id", r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "", err)
		return
	}

	dbSplits, err := cfg.db.GetSplitsByTransactionID(r.Context(), pathTransactionID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not get splits associated with transaction", err)
	}

	var rspPayload []TransactionSplit
	for _, split := range dbSplits {
		addSplit := TransactionSplit{
			ID:            split.ID,
			TransactionID: split.TransactionID,
			CategoryID:    split.CategoryID,
			Amount:        split.Amount,
		}
		rspPayload = append(rspPayload, addSplit)
	}

	respondWithJSON(w, http.StatusOK, rspPayload)
}

func (cfg *APIConfig) handleGetTransaction(w http.ResponseWriter, r *http.Request) {
	getDetails := strings.HasSuffix(r.URL.String(), "/details")

	pathTransactionID, err := parseUUIDFromPath("transaction_id", r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "", err)
		return
	}

	if !getDetails {
		dbTransaction, err := cfg.db.GetTransactionByID(r.Context(), pathTransactionID)
		if err != nil {
			respondWithError(w, http.StatusNotFound, "could not get transaction", err)
			return
		}

		rspPayload := Transaction{
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

		respondWithJSON(w, http.StatusOK, rspPayload)
		return
	} else {
		detailedTxn, err := cfg.db.GetTransactionDetailsByID(r.Context(), pathTransactionID)
		if err != nil {
			respondWithError(w, http.StatusNotFound, "could not get transaction details", err)
			return
		}

		respSplits := make(map[string]int64)
		{
			data := []byte(detailedTxn.Splits)
			source := (*json.RawMessage)(&data)
			err := json.Unmarshal(*source, &respSplits)
			if err != nil {
				respondWithError(w, http.StatusInternalServerError, "failure unmarshalling transaction splits", err)
				return
			}
		}

		rspPayload := TransactionDetail{
			ID:              detailedTxn.ID,
			TransactionType: detailedTxn.TransactionType,
			TransactionDate: detailedTxn.TransactionDate,
			PayeeName:       detailedTxn.PayeeName,
			BudgetName:      detailedTxn.BudgetName.String,
			AccountName:     detailedTxn.AccountName.String,
			LoggerName:      detailedTxn.LoggerName.String,
			TotalAmount:     detailedTxn.TotalAmount,
			Notes:           detailedTxn.Notes,
			Cleared:         detailedTxn.Cleared,
			Splits:          respSplits,
		}

		respondWithJSON(w, http.StatusOK, rspPayload)
		return
	}
}

func (cfg *APIConfig) handleGetTransactions(w http.ResponseWriter, r *http.Request) {
	getDetails := strings.Contains(r.URL.String(), "/details")

	parsedAccountID, err := parseUUIDFromPath("account_id", r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "", err)
		return
	}
	parsedCategoryID, err := parseUUIDFromPath("category_id", r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "", err)
		return
	}
	parsedPayeeID, err := parseUUIDFromPath("payee_id", r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "", err)
		return
	}

	parsedStartDate, err := parseDateFromQuery("start_date", r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "", err)
		return
	}
	parsedEndDate, err := parseDateFromQuery("end_date", r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "", err)
		return
	}

	pathBudgetID := getContextKeyValueAsUUID(r.Context(), "budget_id")

	if !getDetails {
		dbTransactions, err := cfg.db.GetTransactions(r.Context(), database.GetTransactionsParams{
			AccountID:  parsedAccountID,
			CategoryID: parsedCategoryID,
			PayeeID:    parsedPayeeID,
			StartDate:  parsedStartDate,
			EndDate:    parsedEndDate,
			BudgetID:   pathBudgetID,
		})
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "could not retrieve transactions", err)
			return
		}

		var transactions []Transaction
		for _, transaction := range dbTransactions {
			transactions = append(transactions, Transaction{
				ID:              transaction.ID,
				CreatedAt:       transaction.CreatedAt,
				UpdatedAt:       transaction.UpdatedAt,
				BudgetID:        transaction.BudgetID,
				LoggerID:        transaction.LoggerID,
				AccountID:       transaction.AccountID,
				TransactionType: transaction.TransactionType,
				TransactionDate: transaction.TransactionDate,
				PayeeID:         transaction.PayeeID,
				Notes:           transaction.Notes,
				Cleared:         transaction.Cleared,
			})
		}

		type rspSchema struct {
			Transactions []Transaction `json:"transactions"`
		}

		rspPayload := rspSchema{
			Transactions: transactions,
		}

		respondWithJSON(w, http.StatusOK, rspPayload)
		return
	} else {

		detailedTxns, err := cfg.db.GetTransactionDetails(r.Context(), database.GetTransactionDetailsParams{
			AccountID:  parsedAccountID,
			CategoryID: parsedCategoryID,
			PayeeID:    parsedPayeeID,
			StartDate:  parsedStartDate,
			EndDate:    parsedEndDate,
			BudgetID:   pathBudgetID,
		})
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "could not retrieve transactions", err)
			return
		}

		var transactions []TransactionDetail
		for _, detailedTxn := range detailedTxns {

			respSplits := make(map[string]int64)
			{
				data := []byte(detailedTxn.Splits)
				source := (*json.RawMessage)(&data)
				err := json.Unmarshal(*source, &respSplits)
				if err != nil {
					respondWithError(w, http.StatusInternalServerError, "failure unmarshalling transaction splits", err)
					return
				}
			}

			transactions = append(transactions, TransactionDetail{
				ID:              detailedTxn.ID,
				TransactionType: detailedTxn.TransactionType,
				TransactionDate: detailedTxn.TransactionDate,
				PayeeName:       detailedTxn.PayeeName,
				BudgetName:      detailedTxn.BudgetName.String,
				AccountName:     detailedTxn.AccountName.String,
				LoggerName:      detailedTxn.LoggerName.String,
				TotalAmount:     detailedTxn.TotalAmount,
				Notes:           detailedTxn.Notes,
				Cleared:         detailedTxn.Cleared,
				Splits:          respSplits,
			})
		}

		type rspSchema struct {
			Transactions []TransactionDetail `json:"transactions"`
		}

		rspPayload := rspSchema{
			Transactions: transactions,
		}

		respondWithJSON(w, http.StatusOK, rspPayload)
		return
	}
}
