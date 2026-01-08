package api

import (
	"net/http"
)

func (cfg *APIConfig) endpDeleteTransaction(w http.ResponseWriter, r *http.Request) {
	pathTransactionID, err := parseUUIDFromPath("transaction_id", r)
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

	// DB TRANSACTION BLOCK
	{
		tx, err := cfg.Pool.Begin(r.Context())
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "", err)
			return
		}
		defer tx.Rollback(r.Context())

		q := cfg.db.WithTx(tx)

		if err = q.DeleteTransaction(r.Context(), pathTransactionID); err != nil {
			if err != nil {
				respondWithError(w, http.StatusInternalServerError, "could not delete transaction", err)
				return
			}
		}
		if checkIsTransfer(dbTransaction.TransactionType) {
			linkedTxn, err := cfg.db.GetLinkedTransaction(r.Context(), pathTransactionID)
			if err != nil {
				respondWithError(w, http.StatusInternalServerError, "could not get corresponding transfer transaction to delete", err)
				return
			}
			if err := q.DeleteTransaction(r.Context(), linkedTxn.ID); err != nil {
				// NOTE: Different meaning of 'transaction', here.
				respondWithError(w, http.StatusInternalServerError, "could not complete database transaction", err)
				return
			}
		}
		if err := tx.Commit(r.Context()); err != nil {
			respondWithError(w, http.StatusInternalServerError, "", err)
			return
		}
	}

	respondWithCode(w, http.StatusNoContent)
}
