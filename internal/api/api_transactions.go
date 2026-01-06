package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/YouWantToPinch/pincher-api/internal/database"
)

type LogTransactionrqSchema struct {
	AccountName         string `json:"account_name"`
	TransferAccountName string `json:"transfer_account_name"`
	// TransactionDate is a time string in the custom format"2006-01-02" (YYYY-MM-DD)
	TransactionDate string `json:"transaction_date"`
	PayeeName       string `json:"payee_name"`
	Notes           string `json:"notes"`
	Cleared         bool   `json:"is_cleared"`
	/* Amounts is a map of category UUID strings to integers.
	If there is only one entry in Amounts, the transaction is not truly split.
	Nonetheless, all transactions record at least one corresponding split.
	A 'split' reflects the sum of spending toward one particular category within the transaction.
	*/
	Amounts map[string]int64 `json:"amounts"`
}

// Parses relevant input amounts, txnType, txnDate, and transfer status, or returns an error.
// Any txn with no amount, or with amounts not matching in type, are rejected.
func validateTxn(rqPayload *LogTransactionrqSchema) (amounts map[string]int64, txnType string, txnDate time.Time, isTransfer bool, err error) {
	txnDate, err = time.Parse("2006-01-02", rqPayload.TransactionDate)
	if err != nil {
		return nil, "NONE", time.Time{}, false, err
	}

	isTransfer = (rqPayload.TransferAccountName != "")
	txnType = "NONE"

	setTxnType := func(ptr *string, val string) error {
		switch *ptr {
		case "NONE":
			*ptr = val
			return nil
		case val:
			return nil
		default:
			return fmt.Errorf("one or more splits do not match expected type %v", *ptr)
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
			return nil, "NONE", txnDate, false, err
		}
	}
	// return error on txn amount of 0
	if len(parsedAmounts) == 0 {
		return nil, "NONE", txnDate, false, fmt.Errorf("no amount values provided for transaction")
	}
	// sanity check
	if txnType == "NONE" {
		return nil, txnType, txnDate, isTransfer, fmt.Errorf("found one or more amounts in txn, but could not interpret txn type (THIS SHOULD NEVER HAPPEN!)")
	}

	return parsedAmounts, txnType, txnDate, isTransfer, nil
}

func (cfg *APIConfig) endpLogTransaction(w http.ResponseWriter, r *http.Request) {
	rqPayload, err := decodePayload[LogTransactionrqSchema](r)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "", err)
		return
	}

	parsedAmounts, txnType, txnDate, isTransfer, err := validateTxn(&rqPayload)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "could not validate transaction", err)
	}

	pathBudgetID := getContextKeyValue(r.Context(), "budget_id")

	AccountID, err := lookupResourceIDByName(r.Context(),
		database.GetBudgetAccountIDByNameParams{
			AccountName: rqPayload.AccountName,
			BudgetID:    pathBudgetID,
		}, cfg.db.GetBudgetAccountIDByName)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "could not get account id", err)
		return
	}

	PayeeID, err := lookupResourceIDByName(r.Context(),
		database.GetBudgetPayeeIDByNameParams{
			PayeeName: rqPayload.PayeeName,
			BudgetID:  pathBudgetID,
		}, cfg.db.GetBudgetPayeeIDByName)
	if err != nil && !isTransfer {
		respondWithError(w, http.StatusBadRequest, "could not get payee id", err)
		return
	}

	amountsJSONBytes, err := json.Marshal(parsedAmounts)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error(), err)
		return
	}

	validatedUserID := getContextKeyValue(r.Context(), "user_id")

	dbTransaction, err := cfg.db.LogTransaction(r.Context(), database.LogTransactionParams{
		BudgetID:        pathBudgetID,
		LoggerID:        validatedUserID,
		AccountID:       *AccountID,
		TransactionType: txnType,
		TransactionDate: txnDate,
		PayeeID:         *PayeeID,
		Notes:           rqPayload.Notes,
		Cleared:         rqPayload.Cleared,
		Amounts:         json.RawMessage(amountsJSONBytes),
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not log transaction", err)
		return
	}

	var transferTransactionID *uuid.UUID
	var TransferAccountID *uuid.UUID
	if isTransfer {
		// parse transfer_account_id
		TransferAccountID, err = lookupResourceIDByName(r.Context(),
			database.GetBudgetAccountIDByNameParams{
				AccountName: rqPayload.TransferAccountName,
				BudgetID:    pathBudgetID,
			}, cfg.db.GetBudgetAccountIDByName)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "could not get transfer account id", err)
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
			AccountID:       *TransferAccountID,
			TransactionType: invertTransferType(txnType),
			TransactionDate: txnDate,
			PayeeID:         *PayeeID,
			Notes:           rqPayload.Notes,
			Cleared:         rqPayload.Cleared,
			Amounts:         json.RawMessage(invertedAmountsJSONBytes),
		})
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "could not log corresponding transfer transaction", err)
			return
		}
		transferTransactionID = &transferTransaction.ID
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
			respondWithError(w, http.StatusInternalServerError, "could not link transfer transactions", err)
			return
		}
	}

	getTransactionView := func(transactionToViewID uuid.UUID) (TransactionView, error) {
		viewTransaction, err := cfg.db.GetTransactionDetailsByID(r.Context(), transactionToViewID)
		if err != nil {
			return TransactionView{}, fmt.Errorf("could not get transaction from view using id %v: %v", transactionToViewID.String(), err.Error())
		}

		respSplits := make(map[string]int)
		{
			data := []byte(viewTransaction.Splits)
			source := (*json.RawMessage)(&data)
			err := json.Unmarshal(*source, &respSplits)
			if err != nil {
				return TransactionView{}, fmt.Errorf("failure unmarshalling transaction splits: %s", err.Error())
			}
		}

		return TransactionView{
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
		t2, err := getTransactionView(*transferTransactionID)
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
		respondWithError(w, http.StatusBadRequest, "could not get transaction view", err)
		return
	}

	respondWithJSON(w, http.StatusCreated, rspPayload)
}

func (cfg *APIConfig) endpGetTransactions(w http.ResponseWriter, r *http.Request) {
	getDetails := strings.Contains(r.URL.String(), "/details")

	var err error

	var parsedAccountID uuid.UUID
	err = parseUUIDFromPath("account_id", r, &parsedAccountID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "", err)
		return
	}
	var parsedCategoryID uuid.UUID
	err = parseUUIDFromPath("category_id", r, &parsedCategoryID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "", err)
		return
	}
	var parsedPayeeID uuid.UUID
	err = parseUUIDFromPath("payee_id", r, &parsedPayeeID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "", err)
		return
	}

	var parsedStartDate time.Time
	err = parseDateFromQuery("start_date", r, &parsedStartDate)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "", err)
		return
	}
	var parsedEndDate time.Time
	err = parseDateFromQuery("end_date", r, &parsedEndDate)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "", err)
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

		// slog.Debug(fmt.Sprintf("TRANSACTIONS FOUND: %d", len(rspPayload.Transactions)))

		respondWithJSON(w, http.StatusOK, rspPayload)
		return
	} else {

		viewTransactions, err := cfg.db.GetTransactionDetails(r.Context(), database.GetTransactionDetailsParams{
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

		var transactions []TransactionView
		for _, viewTransaction := range viewTransactions {

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
	checkIsTransfer := func(txnType string) bool {
		return txnType == "TRANSFER_TO" || txnType == "TRANSFER_FROM"
	}

	rqPayload, err := decodePayload[LogTransactionrqSchema](r)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "", err)
		return
	}

	parsedAmounts, txnType, txnDate, isTransfer, err := validateTxn(&rqPayload)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "could not validate transaction", err)
	}

	var pathTransactionID uuid.UUID
	err = parseUUIDFromPath("transaction_id", r, &pathTransactionID)
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
	if isTransfer != checkIsTransfer(dbTransaction.TransactionType) {
		respondWithError(w, http.StatusBadRequest, "cannot change transfer txn to non-transfer txn, nor vice-versa", nil)
	}

	pathBudgetID := getContextKeyValue(r.Context(), "budget_id")

	AccountID, err := lookupResourceIDByName(r.Context(),
		database.GetBudgetAccountIDByNameParams{
			AccountName: rqPayload.AccountName,
			BudgetID:    pathBudgetID,
		}, cfg.db.GetBudgetAccountIDByName)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "could not get account ID", err)
		return
	}

	PayeeID, err := lookupResourceIDByName(r.Context(),
		database.GetBudgetPayeeIDByNameParams{
			PayeeName: rqPayload.PayeeName,
			BudgetID:  pathBudgetID,
		}, cfg.db.GetBudgetPayeeIDByName)
	if err != nil && !isTransfer {
		respondWithError(w, http.StatusBadRequest, "could not get payee ID", err)
		return
	}

	amountsJSONBytes, err := json.Marshal(parsedAmounts)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error(), err)
		return
	}

	_, err = cfg.db.UpdateTransaction(r.Context(), database.UpdateTransactionParams{
		TransactionID:   pathTransactionID,
		AccountID:       *AccountID,
		TransactionType: txnType,
		TransactionDate: txnDate,
		PayeeID:         *PayeeID,
		Notes:           rqPayload.Notes,
		Cleared:         rqPayload.Cleared,
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
	pathBudgetID := getContextKeyValue(r.Context(), "budget_id")
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
