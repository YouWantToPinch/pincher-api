package api

import (
	"net/http"

	"github.com/YouWantToPinch/pincher-api/internal/database"
	"github.com/google/uuid"
)

func (cfg *APIConfig) endpUpdateTransaction(w http.ResponseWriter, r *http.Request) {
	validatedTxn := getContextKeyValueAsTxn(r.Context(), "validated_txn")

	pathTransactionID, err := parseUUIDFromPath("transaction_id", r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "", err)
		return
	}

	dbTxnDetails, err := cfg.db.GetTransactionDetailsByID(r.Context(), pathTransactionID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "could not get transaction", err)
		return
	}
	// Non-transfer TXNs may not be updated as transfer TXNs, and vice versa
	if validatedTxn.isTransfer != checkIsTransfer(dbTxnDetails.TransactionType) {
		respondWithError(w, http.StatusBadRequest, "cannot change transfer txn to non-transfer txn, nor vice-versa", nil)
		return
	}

	// DB TRANSACTION BLOCK
	{
		tx, err := cfg.Pool.Begin(r.Context())
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "", err)
			return
		}
		defer tx.Rollback(r.Context())

		q := cfg.db.WithTx(tx)

		splits := map[string]int64{}
		// if amount values did not change, splits may remain untouched
		if dbTxnDetails.TotalAmount != totalFromAmountsMap(validatedTxn.amounts) {
			splits = validatedTxn.amounts
		}
		if msg, err := pgxUpdateTxn(q, r.Context(), database.UpdateTransactionParams{
			TransactionID:   pathTransactionID,
			AccountID:       validatedTxn.accountID,
			TransactionType: validatedTxn.txnType,
			TransactionDate: validatedTxn.txnDate,
			PayeeID:         uuid.Nil,
			Notes:           validatedTxn.notes,
			Cleared:         validatedTxn.cleared,
		}, splits); err != nil {
			errMsgPrefix := "could not update transaction"
			if validatedTxn.isTransfer {
				errMsgPrefix = "could not update transfer transaction"
			}
			respondWithError(w, http.StatusInternalServerError, errMsgPrefix+": "+msg, err)
			return
		}
		if validatedTxn.isTransfer {
			linkedTxn, err := q.GetLinkedTransaction(r.Context(), pathTransactionID)
			if err != nil {
				respondWithError(w, http.StatusInternalServerError, "could not find corresponding txn to update", err)
				return
			}
			if msg, err := pgxUpdateTxn(q, r.Context(), database.UpdateTransactionParams{
				TransactionID:   linkedTxn.ID,
				AccountID:       linkedTxn.AccountID,
				TransactionType: invertTransferType(validatedTxn.txnType),
				TransactionDate: validatedTxn.txnDate,
				PayeeID:         validatedTxn.payeeID,
				Notes:           validatedTxn.notes,
				Cleared:         linkedTxn.Cleared,
			}, invertAmountsMap(splits)); err != nil {
				respondWithError(w, http.StatusInternalServerError, "could not update corresponding transfer transaction: "+msg, err)
				return
			}
		}
		if err := tx.Commit(r.Context()); err != nil {
			// NOTE: Different meaning of 'transaction', here.
			respondWithError(w, http.StatusInternalServerError, "could not complete database transaction", err)
			return
		}
	}

	respondWithCode(w, http.StatusNoContent)
}
