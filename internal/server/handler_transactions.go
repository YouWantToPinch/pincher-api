package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/YouWantToPinch/pincher-api/internal/database"
)

func abs(i int64) int64 {
	if i < 0 {
		return -1 * i
	}
	return i
}

func signByType(transactionType string, i int64) (int64, error) {
	switch transactionType {
	case "TRANSFER_TO", "DEPOSIT":
		return (abs(i)), nil
	case "TRANSFER_FROM", "WITHDRAWAL":
		return -1 * (abs(i)), nil
	default:
		return i, (fmt.Errorf("transaction type \"%s\" not supported", transactionType))
	}
}

func (cfg *apiConfig) endpLogTransaction(w http.ResponseWriter, r *http.Request) {

	type parameters struct {
		AccountID         string    `json:"account_id"`
		TransferAccountID string    `json:"transfer_account_id"`
		TransactionType   string    `json:"transaction_type"`
		TransactionDate   time.Time `json:"transaction_date"`
		PayeeID           string    `json:"payee_id"`
		Notes             string    `json:"notes"`
		Cleared           string    `json:"is_cleared"`
		/* Map of category UUID strings to integers.
		   If there is > 1 entry in Amounts, the transaction is not truly split.
		   Nonetheless, all transactions record at least one corresponding split.
		   A 'split' reflects the sum of spending toward one particular category within the transaction.
		*/
		Amounts map[string]int64 `json:"amounts"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failure decoding parameters when logging transaction", err)
		return
	}

	isTransfer := params.TransactionType == "TRANSFER_TO" || params.TransactionType == "TRANSFER_FROM"

	parsedAccountID, err := uuid.Parse(params.AccountID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Provided account_id string could not be parsed as UUID", err)
		return
	}
	parsedPayeeID, err := uuid.Parse(params.PayeeID)
	if err != nil && !isTransfer {
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
		respondWithError(w, http.StatusBadRequest, "Provided string value for 'Cleared' could not be parsed; must be 'true' or 'false'", nil)
		return
	}

	parsedAmounts := make(map[string]int64)
	for k, v := range params.Amounts {
		parsedAmounts[k], err = signByType(params.TransactionType, v)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, err.Error(), err)
			return
		}
	}
	amountsJsonBytes, err := json.Marshal(parsedAmounts)
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
		TransactionType: params.TransactionType,
		TransactionDate: params.TransactionDate,
		PayeeID:         parsedPayeeID,
		Notes:           params.Notes,
		Cleared:         parsedCleared,
		Amounts:         json.RawMessage(amountsJsonBytes),
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't log transaction", err)
		return
	}

	var transferTransactionID uuid.UUID
	var parsedTransferAccountID uuid.UUID
	if isTransfer {
		// parse transfer_account_id
		parsedTransferAccountID, err = uuid.Parse(params.TransferAccountID)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Provided transfer_account_id string could not be parsed as UUID", err)
			return
		}
		// prepare inverse amounts for corresponding transaction
		invertedAmounts := make(map[string]int64)
		for k, v := range parsedAmounts {
			invertedAmounts[k] = -1 * v
		}
		invertedAmountsJsonBytes, err := json.Marshal(invertedAmounts)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error(), err)
			return
		}
		getOppositeType := func(s string) string {
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
			TransactionType: getOppositeType(params.TransactionType),
			TransactionDate: params.TransactionDate,
			PayeeID:         parsedPayeeID,
			Notes:           params.Notes,
			Cleared:         parsedCleared,
			Amounts:         json.RawMessage(invertedAmountsJsonBytes),
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
			return TransactionView{}, fmt.Errorf("Couldn't get transaction from view using id %v; %v", transactionToViewID.String(), err.Error())
		}

		respSplits := make(map[string]int)
		{
			data := []byte(viewTransaction.Splits)
			source := (*json.RawMessage)(&data)
			err := json.Unmarshal(*source, &respSplits)
			if err != nil {
				return TransactionView{}, errors.New("Failure unmarshalling transaction splits into map[string]int64")
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
		type response struct {
			Transactions []TransactionView `json:"transactions"`
		}
		var respBody response
		t1, err := getTransactionView(dbTransaction.ID)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, err.Error(), err)
			return
		}
		t2, err := getTransactionView(transferTransactionID)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, err.Error(), err)
			return
		}

		respBody.Transactions = append(respBody.Transactions, t1)
		respBody.Transactions = append(respBody.Transactions, t2)

		respondWithJSON(w, http.StatusCreated, respBody)
		return
	}

	respBody, err := getTransactionView(dbTransaction.ID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error(), err)
		return
	}

	respondWithJSON(w, http.StatusCreated, respBody)
}

// Try to parse input path parameter; store uuid.Nil into 'parse' on failure
func parseUUIDFromPath(pathParam string, r *http.Request, parse *uuid.UUID) error {
	idString := r.PathValue(pathParam)
	if idString != "" {
		parsedID, err := uuid.Parse(idString)
		if err != nil {
			return fmt.Errorf("Parameter value '%s' for provided path parameter '%s' could not be parsed as UUID", idString, pathParam)
		}
		*parse = parsedID
	} else {
		*parse = uuid.Nil
	}
	return nil
}

// Try to parse input query parameter; store time.Time{} into 'parse' on failure
func parseDateFromQuery(queryParam string, r *http.Request, parse *time.Time) error {
	const timeLayout = time.RFC3339
	dateString := r.URL.Query().Get(queryParam)
	if dateString != "" {
		parsedDate, err := time.Parse(timeLayout, dateString)
		if err != nil {
			return fmt.Errorf("Query value '%s' for provided parameter '%s' could not be parsed as UUID", dateString, queryParam)
		}
		*parse = parsedDate
	} else {
		*parse = time.Time{}
	}
	return nil
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
	slog.Debug("pathBudgetID: " + pathBudgetID.String())

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

		var respBody []Transaction
		for _, transaction := range dbTransactions {
			addTransaction := Transaction{
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
			}
			respBody = append(respBody, addTransaction)
		}

		slog.Debug(fmt.Sprintf("TRANSACTIONS FOUND: %d", len(respBody)))

		respondWithJSON(w, http.StatusOK, respBody)
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

		var respBody []TransactionView
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

			addTransaction := TransactionView{
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
			}
			respBody = append(respBody, addTransaction)
		}

		slog.Debug(fmt.Sprintf("TRANSACTIONS FOUND: %d", len(respBody)))

		respondWithJSON(w, http.StatusOK, respBody)
		return
	}

}

func (cfg *apiConfig) endpGetTransactionSplits(w http.ResponseWriter, r *http.Request) {
	idString := r.PathValue("transaction_id")
	pathTransactionID, err := uuid.Parse(idString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid id", err)
		return
	}
	dbSplits, err := cfg.db.GetSplitsByTransactionID(r.Context(), pathTransactionID)

	var respBody []TransactionSplit
	for _, split := range dbSplits {
		addSplit := TransactionSplit{
			ID:            split.ID,
			TransactionID: split.TransactionID,
			CategoryID:    split.CategoryID,
			Amount:        split.Amount,
		}
		respBody = append(respBody, addSplit)
	}

	respondWithJSON(w, http.StatusOK, respBody)
}

func (cfg *apiConfig) endpGetTransaction(w http.ResponseWriter, r *http.Request) {
	idString := r.PathValue("transaction_id")
	pathTransactionID, err := uuid.Parse(idString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid id", err)
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

		respBody := TransactionView{
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

		respondWithJSON(w, http.StatusCreated, respBody)
		return
	}

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
}
