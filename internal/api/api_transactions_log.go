package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/YouWantToPinch/pincher-api/internal/database"
	"github.com/google/uuid"
)

// UpsertTransactionRqSchema represents a raw POST or PUT payload sent by a client.
// It is validated through middleware before then being sent to the proper txn endpoint.
type UpsertTransactionRqSchema struct {
	AccountName         string `json:"account_name"`
	TransferAccountName string `json:"transfer_account_name"`
	// TransactionDate is a time string in the custom format "2006-01-02" (YYYY-MM-DD)
	TransactionDate string `json:"transaction_date"`
	PayeeName       string `json:"payee_name"`
	Notes           string `json:"notes"`
	Cleared         bool   `json:"is_cleared"`
	/* Amounts associates category names with the amount spent from each.
	If there is only one entry in Amounts, the transaction is not truly split.
	Nonetheless, all transactions are associated with at least one txn split.
	A 'txn split' reflects the sum of spending toward one particular category within the transaction.
	*/
	Amounts map[string]int64 `json:"amounts"`
}

// validatedTxnPayload represents a validated Upsert request payload.
type validatedTxnPayload struct {
	accountID         uuid.UUID
	transferAccountID uuid.UUID
	payeeID           uuid.UUID
	txnType           string
	txnDate           time.Time
	isTransfer        bool
	amounts           map[string]int64
	notes             string
	cleared           bool
}

func (cfg *APIConfig) endpLogTransaction(w http.ResponseWriter, r *http.Request) {
	validatedTxn := getContextKeyValueAsTxn(r.Context(), "validated_txn")
	pathBudgetID := getContextKeyValueAsUUID(r.Context(), "budget_id")

	amountsJSONBytes, err := json.Marshal(validatedTxn.amounts)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not marshal txn splits", err)
		return
	}

	validatedUserID := getContextKeyValueAsUUID(r.Context(), "user_id")

	dbTxn, err := cfg.db.LogTransaction(r.Context(), database.LogTransactionParams{
		BudgetID:        pathBudgetID,
		LoggerID:        validatedUserID,
		AccountID:       validatedTxn.accountID,
		TransactionType: validatedTxn.txnType,
		TransactionDate: validatedTxn.txnDate,
		PayeeID:         validatedTxn.payeeID,
		Amounts:         json.RawMessage(amountsJSONBytes),
		Notes:           validatedTxn.notes,
		Cleared:         validatedTxn.cleared,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not log transaction", err)
		return
	}

	var transferTxnID *uuid.UUID
	if validatedTxn.isTransfer {
		// prepare inverse amounts for corresponding transaction
		invertedAmounts := make(map[string]int64)
		for k, v := range validatedTxn.amounts {
			invertedAmounts[k] = -1 * v
		}
		invertedAmountsJSONBytes, err := json.Marshal(invertedAmounts)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error(), err)
			return
		}
		invertTransferType := func(s string) string {
			if s == "TRANSFER_TO" {
				return "TRANSFER_FROM"
			}
			return "TRANSFER_TO"
		}
		// log the corresponding transaction
		transferTxn, err := cfg.db.LogTransaction(r.Context(), database.LogTransactionParams{
			BudgetID:        pathBudgetID,
			LoggerID:        validatedUserID,
			AccountID:       validatedTxn.transferAccountID,
			TransactionType: invertTransferType(validatedTxn.txnType),
			TransactionDate: validatedTxn.txnDate,
			PayeeID:         validatedTxn.payeeID,
			Amounts:         json.RawMessage(invertedAmountsJSONBytes),
			Notes:           validatedTxn.notes,
			Cleared:         validatedTxn.cleared,
		})
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "could not log corresponding transfer transaction", err)
			return
		}
		transferTxnID = &transferTxn.ID
		// link transfer transactions
		getTransferIDs := func(t1, t2 *database.LogTransactionRow) (*database.LogTransactionRow, *database.LogTransactionRow) {
			if (*t1).TransactionType == "TRANSFER_FROM" {
				return t2, t1
			} else {
				return t1, t2
			}
		}
		toPtr, fromPtr := getTransferIDs(&dbTxn, &transferTxn)
		_, err = cfg.db.LogAccountTransfer(r.Context(), database.LogAccountTransferParams{
			FromTransactionID: (*fromPtr).ID,
			ToTransactionID:   (*toPtr).ID,
		})
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "could not link transfer transactions", err)
			return
		}
	}

	getTxnDetails := func(txnID uuid.UUID) (*TransactionView, error) {
		viewTransaction, err := cfg.db.GetTransactionDetailsByID(r.Context(), txnID)
		if err != nil {
			return nil, fmt.Errorf("could not get transaction details: %w", err)
		}

		respSplits := make(map[string]int)
		{
			data := []byte(viewTransaction.Splits)
			source := (*json.RawMessage)(&data)
			err := json.Unmarshal(*source, &respSplits)
			if err != nil {
				return nil, fmt.Errorf("failure unmarshalling transaction splits: %w", err)
			}
		}

		return &TransactionView{
			ID:              viewTransaction.ID,
			TransactionType: viewTransaction.TransactionType,
			TransactionDate: viewTransaction.TransactionDate,
			Payee:           viewTransaction.Payee,
			TotalAmount:     viewTransaction.TotalAmount,
			Notes:           viewTransaction.Notes,
			Cleared:         viewTransaction.Cleared,
			Splits:          respSplits,
		}, nil
	}

	if validatedTxn.isTransfer {
		type rspSchema struct {
			Transaction         TransactionView `json:"to_transaction"`
			TransferTransaction TransactionView `json:"transfer_transaction"`
		}
		var rspPayload rspSchema
		t1, err := getTxnDetails(dbTxn.ID)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "", err)
			return
		}
		t2, err := getTxnDetails(*transferTxnID)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "", err)
			return
		}

		rspPayload.Transaction = *t1
		rspPayload.TransferTransaction = *t2

		respondWithJSON(w, http.StatusCreated, rspPayload)
		return
	} else {
		type rspSchema struct {
			TransactionView
		}
		var rspPayload rspSchema
		t, err := getTxnDetails(dbTxn.ID)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "", err)
			return
		}
		rspPayload.TransactionView = *t

		respondWithJSON(w, http.StatusCreated, rspPayload)
		return
	}
}
