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

	validatedUserID := getContextKeyValueAsUUID(r.Context(), "user_id")

	// DB TRANSACTION BLOCK
	{
		tx, err := cfg.Pool.Begin(r.Context())
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "", err)
			return
		}
		defer tx.Rollback(r.Context())

		q := cfg.db.WithTx(tx)

		// getTxnDetails can be called on an ID for a return of its user-friendly
		// values in a subsequent response
		getTxnDetails := func(txnID uuid.UUID) (*TransactionDetail, error) {
			detailedTxn, err := q.GetTransactionDetailsByID(r.Context(), txnID)
			if err != nil {
				return nil, fmt.Errorf("could not get transaction details: %w", err)
			}

			respSplits := make(map[string]int64)
			{
				data := []byte(detailedTxn.Splits)
				source := (*json.RawMessage)(&data)
				err := json.Unmarshal(*source, &respSplits)
				if err != nil {
					return nil, fmt.Errorf("could not unmarshal splits: %w", err)
				}
			}

			return &TransactionDetail{
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
			}, nil
		}

		newTxn, msg, err := pgxLogTxn(q, r.Context(), database.LogTransactionParams{
			BudgetID:        pathBudgetID,
			LoggerID:        validatedUserID,
			AccountID:       validatedTxn.accountID,
			TransactionType: validatedTxn.txnType,
			TransactionDate: validatedTxn.txnDate,
			PayeeID:         validatedTxn.payeeID,
			Notes:           validatedTxn.notes,
			Cleared:         validatedTxn.cleared,
		}, validatedTxn.amounts)
		if err != nil {
			errMsgPrefix := "could not update transaction"
			if validatedTxn.isTransfer {
				errMsgPrefix = "could not update transfer transaction"
			}
			respondWithError(w, http.StatusInternalServerError, errMsgPrefix+": "+msg, err)
			return
		}
		if validatedTxn.isTransfer {
			// log the corresponding transaction
			transferTxn, msg, err := pgxLogTxn(q, r.Context(), database.LogTransactionParams{
				BudgetID:        pathBudgetID,
				LoggerID:        validatedUserID,
				AccountID:       validatedTxn.transferAccountID,
				TransactionType: validatedTxn.txnType,
				TransactionDate: validatedTxn.txnDate,
				PayeeID:         validatedTxn.payeeID,
				Notes:           validatedTxn.notes,
				Cleared:         validatedTxn.cleared,
			}, invertAmountsMap(validatedTxn.amounts))
			if err != nil {
				errMsgPrefix := "could not log transaction"
				if validatedTxn.isTransfer {
					errMsgPrefix = "could not log corresponding transfer transaction"
				}
				respondWithError(w, http.StatusInternalServerError, errMsgPrefix+": "+msg, err)
				return
			}
			// link transfer transactions
			toPtr, fromPtr := getTransferIDs(newTxn, transferTxn)
			_, err = q.LogAccountTransfer(r.Context(), database.LogAccountTransferParams{
				FromTransactionID: (*fromPtr).ID,
				ToTransactionID:   (*toPtr).ID,
			})
			if err != nil {
				respondWithError(w, http.StatusInternalServerError, "could not link transfer transactions", err)
				return
			}
			// if a transfer, get the details for both txns logged
			type rspSchema struct {
				FromTransaction TransactionDetail `json:"from_transaction"`
				ToTransaction   TransactionDetail `json:"to_transaction"`
			}
			fromTxnDetails, err := getTxnDetails(fromPtr.ID)
			if err != nil {
				respondWithError(w, http.StatusBadRequest, "", err)
				return
			}
			toTxnDetails, err := getTxnDetails(toPtr.ID)
			if err != nil {
				respondWithError(w, http.StatusBadRequest, "", err)
				return
			}
			rspPayload := rspSchema{
				FromTransaction: *fromTxnDetails,
				ToTransaction:   *toTxnDetails,
			}
			if err := tx.Commit(r.Context()); err != nil {
				respondWithError(w, http.StatusInternalServerError, "", err)
				return
			}
			respondWithJSON(w, http.StatusCreated, rspPayload)
			return

			// if not a transfer, just get the details for txn logged
		} else {
			type rspSchema struct {
				TransactionDetail
			}
			var rspPayload rspSchema
			txnDetails, err := getTxnDetails(newTxn.ID)
			if err != nil {
				respondWithError(w, http.StatusBadRequest, "", err)
				return
			}
			rspPayload.TransactionDetail = *txnDetails

			if err := tx.Commit(r.Context()); err != nil {
				respondWithError(w, http.StatusInternalServerError, "", err)
				return
			}
			respondWithJSON(w, http.StatusCreated, rspPayload)
			return
		}
	}
}
