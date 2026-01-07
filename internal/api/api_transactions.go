package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/YouWantToPinch/pincher-api/internal/database"
)

func (cfg *APIConfig) endpGetTransactionSplits(w http.ResponseWriter, r *http.Request) {
	var pathTransactionID uuid.UUID
	err := parseUUIDFromPath("transaction_id", r, &pathTransactionID)
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

func (cfg *APIConfig) endpGetTransaction(w http.ResponseWriter, r *http.Request) {
	getDetails := strings.HasSuffix(r.URL.String(), "/details")

	var pathTransactionID uuid.UUID
	err := parseUUIDFromPath("transaction_id", r, &pathTransactionID)
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
		viewTransaction, err := cfg.db.GetTransactionDetailsByID(r.Context(), pathTransactionID)
		if err != nil {
			respondWithError(w, http.StatusNotFound, "could not get transaction details", err)
			return
		}

		respSplits := make(map[string]int)
		{
			data := []byte(viewTransaction.Splits)
			source := (*json.RawMessage)(&data)
			err := json.Unmarshal(*source, &respSplits)
			if err != nil {
				respondWithError(w, http.StatusInternalServerError, "failure unmarshalling transaction splits", err)
				return
			}
		}

		rspPayload := TransactionView{
			ID:              viewTransaction.ID,
			TransactionDate: viewTransaction.TransactionDate,
			Payee:           viewTransaction.Payee,
			TotalAmount:     viewTransaction.TotalAmount,
			Notes:           viewTransaction.Notes,
			Cleared:         viewTransaction.Cleared,
			Splits:          respSplits,
		}

		respondWithJSON(w, http.StatusOK, rspPayload)
		return
	}
}

func (cfg *APIConfig) endpUpdateTransaction(w http.ResponseWriter, r *http.Request) {
	validatedTxn := getContextKeyValueAsTxn(r.Context(), "validated_txn")

	checkIsTransfer := func(txnType string) bool {
		return txnType == "TRANSFER_TO" || txnType == "TRANSFER_FROM"
	}

	var pathTransactionID uuid.UUID
	err := parseUUIDFromPath("transaction_id", r, &pathTransactionID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "", err)
		return
	}

	// Non-transfer TXNs may not be updated as transfer TXNs, and vice versa
	dbTransaction, err := cfg.db.GetTransactionByID(r.Context(), pathTransactionID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "could not get transaction", err)
		return
	}
	if validatedTxn.isTransfer != checkIsTransfer(dbTransaction.TransactionType) {
		respondWithError(w, http.StatusBadRequest, "cannot change transfer txn to non-transfer txn, nor vice-versa", nil)
	}

	amountsJSONBytes, err := json.Marshal(validatedTxn.amounts)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not marshal txn splits", err)
		return
	}

	err = cfg.db.UpdateTransaction(r.Context(), database.UpdateTransactionParams{
		TransactionID:   pathTransactionID,
		AccountID:       validatedTxn.accountID,
		TransactionType: validatedTxn.txnType,
		TransactionDate: validatedTxn.txnDate,
		PayeeID:         validatedTxn.payeeID,
		Notes:           validatedTxn.notes,
		Cleared:         validatedTxn.cleared,
		Amounts:         json.RawMessage(amountsJSONBytes),
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not update transaction", err)
		return
	}

	respondWithCode(w, http.StatusNoContent)
}

func (cfg *APIConfig) endpDeleteTransaction(w http.ResponseWriter, r *http.Request) {
	var pathTransactionID uuid.UUID
	err := parseUUIDFromPath("transaction_id", r, &pathTransactionID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "", err)
		return
	}

	dbTransaction, err := cfg.db.GetTransactionByID(r.Context(), pathTransactionID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "could not get transaction", err)
		return
	}
	pathBudgetID := getContextKeyValueAsUUID(r.Context(), "budget_id")
	if pathBudgetID != dbTransaction.BudgetID {
		respondWithCode(w, http.StatusForbidden)
		return
	}

	err = cfg.db.DeleteTransaction(r.Context(), pathTransactionID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not delete transaction", err)
		return
	}

	respondWithCode(w, http.StatusNoContent)
}
