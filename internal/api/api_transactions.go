package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/YouWantToPinch/pincher-api/internal/database"
)

type LogTransactionrqSchema struct {
	AccountID         string    `json:"account_id"`
	TransferAccountID string    `json:"transfer_account_id"`
	TransactionDate   time.Time `json:"transaction_date"`
	PayeeID           string    `json:"payee_id"`
	Notes             string    `json:"notes"`
	Cleared           string    `json:"is_cleared"`
	/* Map of category UUID strings to integers.
	If there is only one entry in Amounts, the transaction is not truly split.
	Nonetheless, all transactions record at least one corresponding split.
	A 'split' reflects the sum of spending toward one particular category within the transaction.
	*/
	Amounts map[string]int64 `json:"amounts"`
}

// Parses relevant input amounts, txnType, transfer status, or returns an error.
// Any txn with no amount, or with amounts not matching in type, are rejected.
func validateTxn(rqPayload *LogTransactionrqSchema) (isCleared bool, amounts map[string]int64, txnType string, isTransfer bool, err error) {
	isCleared, err = parseBoolFromString(rqPayload.Cleared)
	if err != nil {
		return false, nil, "NONE", false, err
	}

	_, transferErr := uuid.Parse(rqPayload.TransferAccountID)
	isTransfer = (transferErr == nil)
	txnType = "NONE"

	setTxnType := func(ptr *string, val string) error {
		switch *ptr {
		case "NONE":
			*ptr = val
			return nil
		case val:
			return nil
		default:
			return errors.New("one or more splits do not match expected type: " + *ptr)
		}
	}

	parsedAmounts := rqPayload.Amounts
	for k, v := range rqPayload.Amounts {
		switch {
		case v > 0:
			if isTransfer {
				err = setTxnType(&txnType, "TRANSFER_TO")
			} else {
				err = setTxnType(&txnType, "DEPOSIT")
			}
		case v < 0:
			if isTransfer {
				err = setTxnType(&txnType, "TRANSFER_FROM")
			} else {
				err = setTxnType(&txnType, "WITHDRAWAL")
			}
		default:
			delete(parsedAmounts, k)
		}
		// return error on txnType mismatch
		if err != nil {
			return isCleared, nil, "NONE", false, err
		}
	}
	// return error on txn amount of 0
	if len(parsedAmounts) == 0 {
		return isCleared, nil, "NONE", false, errors.New("no amount values provided for transaction")
	}
	// sanity check
	if txnType == "NONE" {
		return isCleared, nil, txnType, isTransfer, errors.New("found one or more amounts in txn, but could not interpret txn type (THIS SHOULD NEVER HAPPEN!)")
	}

	return isCleared, parsedAmounts, txnType, isTransfer, nil
}

func (cfg *apiConfig) endpLogTransaction(w http.ResponseWriter, r *http.Request) {
	rqPayload, err := decodePayload[LogTransactionrqSchema](r)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failure decoding parameters when logging transaction", err)
		return
	}

	parsedCleared, parsedAmounts, txnType, isTransfer, err := validateTxn(&rqPayload)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Failure validating transaction", err)
	}

	parsedAccountID, err := uuid.Parse(rqPayload.AccountID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Provided account_id string could not be parsed as UUID", err)
		return
	}
	parsedPayeeID, err := uuid.Parse(rqPayload.PayeeID)
	if err != nil && !isTransfer {
		respondWithError(w, http.StatusBadRequest, "Provided payee_id string could not be parsed as UUID", err)
		return
	}

	amountsJSONBytes, err := json.Marshal(parsedAmounts)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error(), err)
		return
	}

	validatedUserID := getContextKeyValue(r.Context(), "user_id")
	pathBudgetID := getContextKeyValue(r.Context(), "budget_id")

	dbTransaction, err := cfg.db.LogTransaction(r.Context(), database.LogTransactionParams{
		BudgetID:        pathBudgetID,
		LoggerID:        validatedUserID,
		AccountID:       parsedAccountID,
		TransactionType: txnType,
		TransactionDate: rqPayload.TransactionDate,
		PayeeID:         parsedPayeeID,
		Notes:           rqPayload.Notes,
		Cleared:         parsedCleared,
		Amounts:         json.RawMessage(amountsJSONBytes),
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't log transaction", err)
		return
	}

	var transferTransactionID uuid.UUID
	var parsedTransferAccountID uuid.UUID
	if isTransfer {
		// parse transfer_account_id
		parsedTransferAccountID, err = uuid.Parse(rqPayload.TransferAccountID)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Provided transfer_account_id string could not be parsed as UUID", err)
			return
		}
		// prepare inverse amounts for corresponding transaction
		invertedAmounts := make(map[string]int64)
		for k, v := range parsedAmounts {
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
		transferTransaction, err := cfg.db.LogTransaction(r.Context(), database.LogTransactionParams{
			BudgetID:        pathBudgetID,
			LoggerID:        validatedUserID,
			AccountID:       parsedTransferAccountID,
			TransactionType: invertTransferType(txnType),
			TransactionDate: rqPayload.TransactionDate,
			PayeeID:         parsedPayeeID,
			Notes:           rqPayload.Notes,
			Cleared:         parsedCleared,
			Amounts:         json.RawMessage(invertedAmountsJSONBytes),
		})
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Couldn't log corresponding transfer transaction", err)
			return
		}
		transferTransactionID = transferTransaction.ID
		// link transfer transactions
		getTransferIDs := func(t1, t2 *database.LogTransactionRow) (*database.LogTransactionRow, *database.LogTransactionRow) {
			if (*t1).TransactionType == "TRANSFER_FROM" {
				return t2, t1
			} else {
				return t1, t2
			}
		}
		toPtr, fromPtr := getTransferIDs(&dbTransaction, &transferTransaction)
		_, err = cfg.db.LogAccountTransfer(r.Context(), database.LogAccountTransferParams{
			FromTransactionID: (*fromPtr).ID,
			ToTransactionID:   (*toPtr).ID,
		})
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Couldn't link transfer transactions", err)
			return
		}
	}

	getTransactionView := func(transactionToViewID uuid.UUID) (TransactionView, error) {
		viewTransaction, err := cfg.db.GetTransactionFromViewByID(r.Context(), transactionToViewID)
		if err != nil {
			return TransactionView{}, fmt.Errorf("couldn't get transaction from view using id %v; %v", transactionToViewID.String(), err.Error())
		}

		respSplits := make(map[string]int)
		{
			data := []byte(viewTransaction.Splits)
			source := (*json.RawMessage)(&data)
			err := json.Unmarshal(*source, &respSplits)
			if err != nil {
				return TransactionView{}, errors.New("failure unmarshalling transaction splits into map[string]int64")
			}
		}

		return TransactionView{
			ID:              viewTransaction.ID,
			BudgetID:        viewTransaction.BudgetID,
			LoggerID:        viewTransaction.LoggerID,
			AccountID:       viewTransaction.AccountID,
			TransactionType: viewTransaction.TransactionType,
			TransactionDate: viewTransaction.TransactionDate,
			Payee:           viewTransaction.Payee,
			PayeeID:         viewTransaction.PayeeID,
			TotalAmount:     viewTransaction.TotalAmount,
			Notes:           viewTransaction.Notes,
			Cleared:         viewTransaction.Cleared,
			Splits:          respSplits,
		}, nil
	}

	if isTransfer {
		type rspSchema struct {
			Transactions []TransactionView `json:"transactions"`
		}
		var rspPayload rspSchema
		t1, err := getTransactionView(dbTransaction.ID)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "", err)
			return
		}
		t2, err := getTransactionView(transferTransactionID)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "", err)
			return
		}

		rspPayload.Transactions = append(rspPayload.Transactions, t1)
		rspPayload.Transactions = append(rspPayload.Transactions, t2)

		respondWithJSON(w, http.StatusCreated, rspPayload)
		return
	}

	rspPayload, err := getTransactionView(dbTransaction.ID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "", err)
		return
	}

	respondWithJSON(w, http.StatusCreated, rspPayload)
}

func (cfg *apiConfig) endpGetTransactions(w http.ResponseWriter, r *http.Request) {
	var err error

	var parsedAccountID uuid.UUID
	err = parseUUIDFromPath("account_id", r, &parsedAccountID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid parameter value", err)
		return
	}
	var parsedCategoryID uuid.UUID
	err = parseUUIDFromPath("category_id", r, &parsedCategoryID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid parameter value", err)
		return
	}
	var parsedPayeeID uuid.UUID
	err = parseUUIDFromPath("payee_id", r, &parsedPayeeID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid parameter value", err)
		return
	}

	var parsedStartDate time.Time
	err = parseDateFromQuery("start_date", r, &parsedStartDate)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid parameter value", err)
		return
	}
	var parsedEndDate time.Time
	err = parseDateFromQuery("end_date", r, &parsedEndDate)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid parameter value", err)
		return
	}

	pathBudgetID := getContextKeyValue(r.Context(), "budget_id")

	slog.Debug(fmt.Sprintf("Transaction paramaters: budget_id=%s, account_id=%v (nil=%v), category_id=%v (nil=%v), payee_id=%v (nil=%v), start_date=%v (zero=%v), end_date=%v (zero=%v)",
		pathBudgetID.String(),
		parsedAccountID,
		parsedAccountID == uuid.Nil,
		parsedCategoryID,
		parsedCategoryID == uuid.Nil,
		parsedPayeeID,
		parsedPayeeID == uuid.Nil,
		parsedStartDate,
		parsedStartDate.IsZero(),
		parsedEndDate,
		parsedEndDate.IsZero()),
	)

	getMode := r.URL.Query().Get("mode")
	switch getMode {
	case "db":
		dbTransactions, err := cfg.db.GetTransactions(r.Context(), database.GetTransactionsParams{
			AccountID:  parsedAccountID,
			CategoryID: parsedCategoryID,
			PayeeID:    parsedPayeeID,
			StartDate:  parsedStartDate,
			EndDate:    parsedEndDate,
			BudgetID:   pathBudgetID,
		})
		if err != nil {
			respondWithError(w, http.StatusNotFound, "No transactions found", err)
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

		// slog.Debug(fmt.Sprintf("TRANSACTIONS FOUND: %d", len(rspPayload.Transactions)))

		respondWithJSON(w, http.StatusOK, rspPayload)
		return
	default: // or case "view"

		viewTransactions, err := cfg.db.GetTransactionsFromView(r.Context(), database.GetTransactionsFromViewParams{
			AccountID:  parsedAccountID,
			CategoryID: parsedCategoryID,
			PayeeID:    parsedPayeeID,
			StartDate:  parsedStartDate,
			EndDate:    parsedEndDate,
			BudgetID:   pathBudgetID,
		})
		if err != nil {
			respondWithError(w, http.StatusNotFound, "No transactions found", err)
			return
		}

		var transactions []TransactionView
		for _, viewTransaction := range viewTransactions {

			respSplits := make(map[string]int)
			{
				data := []byte(viewTransaction.Splits)
				source := (*json.RawMessage)(&data)
				err := json.Unmarshal(*source, &respSplits)
				if err != nil {
					respondWithError(w, http.StatusInternalServerError, "Failure decoding dbTransaction splits into map[string]int", err)
					return
				}
			}

			transactions = append(transactions, TransactionView{
				ID:              viewTransaction.ID,
				BudgetID:        viewTransaction.BudgetID,
				LoggerID:        viewTransaction.LoggerID,
				AccountID:       viewTransaction.AccountID,
				TransactionType: viewTransaction.TransactionType,
				TransactionDate: viewTransaction.TransactionDate,
				Payee:           viewTransaction.Payee,
				PayeeID:         viewTransaction.PayeeID,
				TotalAmount:     viewTransaction.TotalAmount,
				Notes:           viewTransaction.Notes,
				Cleared:         viewTransaction.Cleared,
				Splits:          respSplits,
			})
		}

		type rspSchema struct {
			Transactions []TransactionView `json:"transactions"`
		}

		rspPayload := rspSchema{
			Transactions: transactions,
		}

		// slog.Debug(fmt.Sprintf("TRANSACTIONS FOUND: %d", len(rspPayload.Transactions)))

		respondWithJSON(w, http.StatusOK, rspPayload)
		return
	}
}

func (cfg *apiConfig) endpGetTransactionSplits(w http.ResponseWriter, r *http.Request) {
	var pathTransactionID uuid.UUID
	err := parseUUIDFromPath("transaction_id", r, &pathTransactionID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid parameter value", err)
		return
	}

	dbSplits, err := cfg.db.GetSplitsByTransactionID(r.Context(), pathTransactionID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Found no splits associated with this transaction", err)
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

func (cfg *apiConfig) endpGetTransaction(w http.ResponseWriter, r *http.Request) {
	var pathTransactionID uuid.UUID
	err := parseUUIDFromPath("transaction_id", r, &pathTransactionID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid parameter value", err)
		return
	}

	getMode := r.URL.Query().Get("mode")
	switch getMode {
	case "db":
		dbTransaction, err := cfg.db.GetTransactionByID(r.Context(), pathTransactionID)
		if err != nil {
			respondWithError(w, http.StatusNotFound, "Couldn't find transaction with specified id", err)
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

		respondWithJSON(w, http.StatusCreated, rspPayload)
		return
	default: // or case "view"
		viewTransaction, err := cfg.db.GetTransactionFromViewByID(r.Context(), pathTransactionID)
		if err != nil {
			respondWithError(w, http.StatusNotFound, "Couldn't find transaction with specified id", err)
			return
		}

		respSplits := make(map[string]int)
		{
			data := []byte(viewTransaction.Splits)
			source := (*json.RawMessage)(&data)
			err := json.Unmarshal(*source, &respSplits)
			if err != nil {
				respondWithError(w, http.StatusInternalServerError, "Failure decoding dbTransaction splits into map[string]int", err)
				return
			}
		}

		rspPayload := TransactionView{
			ID:              viewTransaction.ID,
			BudgetID:        viewTransaction.BudgetID,
			LoggerID:        viewTransaction.LoggerID,
			AccountID:       viewTransaction.AccountID,
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

func (cfg *apiConfig) endpUpdateTransaction(w http.ResponseWriter, r *http.Request) {
	checkIsTransfer := func(txnType string) bool {
		return txnType == "TRANSFER_TO" || txnType == "TRANSFER_FROM"
	}

	rqPayload, err := decodePayload[LogTransactionrqSchema](r)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failure decoding parameters", err)
		return
	}

	parsedCleared, parsedAmounts, txnType, isTransfer, err := validateTxn(&rqPayload)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Failure validating transaction", err)
	}

	var pathTransactionID uuid.UUID
	err = parseUUIDFromPath("transaction_id", r, &pathTransactionID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid id", err)
		return
	}

	// Non-transfer TXNs may not be updated as transfer TXNs, and vice versa
	dbTransaction, err := cfg.db.GetTransactionByID(r.Context(), pathTransactionID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Couldn't find transaction with specified id", err)
		return
	}
	if isTransfer != checkIsTransfer(dbTransaction.TransactionType) {
		respondWithError(w, http.StatusBadRequest, "Transaction type cannot be changed!", nil)
	}

	parsedAccountID, err := uuid.Parse(rqPayload.AccountID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Provided account_id string could not be parsed as UUID", err)
		return
	}
	parsedPayeeID, err := uuid.Parse(rqPayload.PayeeID)
	if err != nil && !isTransfer {
		respondWithError(w, http.StatusBadRequest, "Provided payee_id string could not be parsed as UUID", err)
		return
	}

	amountsJSONBytes, err := json.Marshal(parsedAmounts)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error(), err)
		return
	}

	_, err = cfg.db.UpdateTransaction(r.Context(), database.UpdateTransactionParams{
		TransactionID:   pathTransactionID,
		AccountID:       parsedAccountID,
		TransactionType: txnType,
		TransactionDate: rqPayload.TransactionDate,
		PayeeID:         parsedPayeeID,
		Notes:           rqPayload.Notes,
		Cleared:         parsedCleared,
		Amounts:         json.RawMessage(amountsJSONBytes),
	})
	if err != nil {
		respondWithError(w, http.StatusNotModified, "Failed to update transaction", err)
		return
	}

	respondWithText(w, http.StatusNoContent, "Transaction updated successfully!")
}

func (cfg *apiConfig) endpDeleteTransaction(w http.ResponseWriter, r *http.Request) {
	var pathTransactionID uuid.UUID
	err := parseUUIDFromPath("transaction_id", r, &pathTransactionID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid parameter value", err)
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
}
