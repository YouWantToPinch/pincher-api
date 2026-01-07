package api

import (
	"encoding/json"
	"net/http"

	"github.com/YouWantToPinch/pincher-api/internal/database"
	"github.com/google/uuid"
)

func (cfg *APIConfig) ZendpUpdateTransaction(w http.ResponseWriter, r *http.Request) {
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

	// avoid update to transfer if the linked transfer cannot be found
	var linkedTxn database.Transaction
	var updateLinkedTxn bool
	var invertedAmountsJSONBytes []byte
	if validatedTxn.isTransfer {
		// linkedTxn, err = cfg.db.GetLinkedTransactionID(r.Context(), pathTransactionID)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "could not find corresponding txn to update", err)
			return
		}
		// only changes to certain fields warrant further queries to keep the transfer in sync
		updateLinkedTxn = false /* (linkedTxn.Notes != validatedTxn.notes) ||
		(linkedTxn.TransactionType == validatedTxn.txnType) || */
		// prepare inverse amounts for corresponding transfer transaction
		invertedAmounts := make(map[string]int64)
		for k, v := range validatedTxn.amounts {
			invertedAmounts[k] = -1 * v
		}
		invertedAmountsJSONBytes, err = json.Marshal(invertedAmounts)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "could not marshal txn splits for corresponding transfer txn", err)
			return
		}
	}

	err = cfg.db.UpdateTransaction(r.Context(), database.UpdateTransactionParams{
		TransactionID:   pathTransactionID,
		AccountID:       validatedTxn.accountID,
		TransactionType: validatedTxn.txnType,
		TransactionDate: validatedTxn.txnDate,
		PayeeID:         uuid.Nil,
		Notes:           validatedTxn.notes,
		Cleared:         linkedTxn.Cleared,
		Amounts:         json.RawMessage(amountsJSONBytes),
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not update transaction", err)
		return
	}

	if validatedTxn.isTransfer && updateLinkedTxn {
		invertTransferType := func(s string) string {
			if s == "TRANSFER_TO" {
				return "TRANSFER_FROM"
			}
			return "TRANSFER_TO"
		}

		err = cfg.db.UpdateTransaction(r.Context(), database.UpdateTransactionParams{
			TransactionID:   linkedTxn.ID,
			AccountID:       validatedTxn.accountID,
			TransactionType: invertTransferType(validatedTxn.txnType),
			TransactionDate: validatedTxn.txnDate,
			PayeeID:         validatedTxn.payeeID,
			Notes:           validatedTxn.notes,
			Cleared:         validatedTxn.cleared,
			Amounts:         json.RawMessage(invertedAmountsJSONBytes),
		})
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "could not update transaction", err)
			return
		}

	}

	respondWithCode(w, http.StatusNoContent)
}
