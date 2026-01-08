package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"time"

	"github.com/YouWantToPinch/pincher-api/internal/database"
	"github.com/google/uuid"
)

// validateTxnInput parses relevant inputs: txn amounts, txnDate, transfer status, txnType.
// Any txn with no amount, or with amounts not matching in type, are rejected.
// Any error returned implies a bad request.
func validateTxnInput(rqPayload *UpsertTransactionRqSchema) (*validatedTxnPayload, error) {
	validatedTxn := &validatedTxnPayload{amounts: map[string]int64{}}
	var err error

	validatedTxn.txnDate, err = time.Parse("2006-01-02", rqPayload.TransactionDate)
	if err != nil {
		return nil, fmt.Errorf("transaction date could not be parsed")
	}

	validatedTxn.isTransfer = (rqPayload.TransferAccountName != "")
	validatedTxn.txnType = "NONE"

	setTxnType := func(ptr *string, val string) error {
		switch *ptr {
		case "NONE":
			*ptr = val
			return nil
		case val:
			return nil
		default:
			return fmt.Errorf("one or more splits do not match expected type '%v'", *ptr)
		}
	}

	maps.Copy(validatedTxn.amounts, rqPayload.Amounts)
	for k, v := range rqPayload.Amounts {
		if k == "" {
			return nil, fmt.Errorf("found missing category name from one or more amount fields")
		}
		switch {
		case v > 0:
			if validatedTxn.isTransfer {
				err = setTxnType(&validatedTxn.txnType, "TRANSFER_TO")
			} else {
				err = setTxnType(&validatedTxn.txnType, "DEPOSIT")
			}
		case v < 0:
			if validatedTxn.isTransfer {
				err = setTxnType(&validatedTxn.txnType, "TRANSFER_FROM")
			} else {
				err = setTxnType(&validatedTxn.txnType, "WITHDRAWAL")
			}
		default:
			delete(validatedTxn.amounts, k)
		}
		// return error on txnType mismatch
		if err != nil {
			return nil, fmt.Errorf("inconsistent signage on amount values")
		}
	}
	// return error on txn amount of 0
	if len(validatedTxn.amounts) == 0 {
		return nil, fmt.Errorf("no non-zero amount specified for transaction")
	}
	return validatedTxn, nil
}

func lookupResourceIDByName[T any](ctx context.Context, arg T, dbQuery func(context.Context, T) (uuid.UUID, error)) (*uuid.UUID, error) {
	id, err := dbQuery(ctx, arg)
	if err != nil {
		return &uuid.Nil, err
	}
	return &id, err
}

func checkIsTransfer(txnType string) bool {
	return txnType == "TRANSFER_TO" || txnType == "TRANSFER_FROM"
}

// invertAmountsMap takes a map of strings to int64s,
// and returns an identical map with the int64 values multiplied by -1.
func invertAmountsMap(input map[string]int64) map[string]int64 {
	inverted := make(map[string]int64)
	for k, v := range input {
		inverted[k] = -1 * v
	}
	return inverted
}

func totalFromAmountsMap(input map[string]int64) int64 {
	var total int64
	for _, v := range input {
		total += v
	}
	return total
}

// invertTransferType returns the counterpart of a given transfer type.
func invertTransferType(s string) string {
	if s == "TRANSFER_TO" {
		return "TRANSFER_FROM"
	}
	return "TRANSFER_TO"
}

// getTransferIDs returns two given LogTransactionRow pointers labelled with their types.
func getTransferIDs(t1, t2 *database.Transaction) (*database.Transaction, *database.Transaction) {
	if (*t1).TransactionType == "TRANSFER_FROM" {
		return t2, t1
	} else {
		return t1, t2
	}
}

// pgxUpdateTxn performs an update on a transaction given the parameters,
// including deletion of existing splits and re-insertion of new ones, if necessary.
func pgxUpdateTxn(q *database.Queries, ctx context.Context, params database.UpdateTransactionParams, splits map[string]int64) (errMsg string, err error) {
	if checkIsTransfer(params.TransactionType) {
		amount := totalFromAmountsMap(splits)
		slog.Debug("TRANSFER TXN", slog.Int64("amount", amount))
	}
	chooseErrMsg := func(onTransferFail, onOther string) string {
		if checkIsTransfer(params.TransactionType) {
			return onTransferFail
		}
		return onOther
	}
	if err = q.UpdateTransaction(ctx, params); err != nil {
		return chooseErrMsg("could not update corresponding transfer transaction",
			"could not update transaction"), nil
	}
	if len(splits) == 0 {
		return "", nil
	} else {
		var amountsJSONBytes []byte
		if amountsJSONBytes, err = json.Marshal(splits); err != nil {
			return chooseErrMsg("could not marshal split for corresponding transfer transaction",
				"could not marshal splits for transaction"), nil
		}
		if err = q.DeleteTransactionSplits(ctx, params.TransactionID); err != nil {
			return chooseErrMsg("could not delete corresponding transfer transaction splits",
				"could not delete transaction splits"), nil
		}
		if _, err = q.LogTransactionSplits(ctx, database.LogTransactionSplitsParams{
			TransactionID: params.TransactionID,
			Amounts:       amountsJSONBytes,
		}); err != nil {
			return chooseErrMsg("could not update corresponding transfer transaction with new splits",
				"could not update transaction with new splits"), nil
		}
	}
	return "", nil
}

// pgxLogTxn performs an insert on the transasction table given the parameters,
// and inserts the relevant transaction splits as well.
func pgxLogTxn(q *database.Queries, ctx context.Context, params database.LogTransactionParams, splits map[string]int64) (txn *database.Transaction, errMsg string, err error) {
	// TODO: Add unit tests for all of this business logic I've been able to separate! :)
	if checkIsTransfer(params.TransactionType) {
		amount := totalFromAmountsMap(splits)
		slog.Debug("TRANSFER TXN", slog.Int64("amount", amount))
	}
	chooseErrMsg := func(onTransferFail, onOther string) string {
		if checkIsTransfer(params.TransactionType) {
			return onTransferFail
		}
		return onOther
	}
	newTxn, err := q.LogTransaction(ctx, params)
	if err != nil {
		return nil, "could not log transaction", err
	}
	if len(splits) == 0 {
		return nil, "", nil
	} else {
		var amountsJSONBytes []byte
		if amountsJSONBytes, err = json.Marshal(splits); err != nil {
			return nil, chooseErrMsg("could not marshal split for corresponding transfer transaction to log",
				"could not marshal splits for new transaction"), nil
		}
		if _, err := q.LogTransactionSplits(ctx, database.LogTransactionSplitsParams{
			TransactionID: newTxn.ID,
			Amounts:       amountsJSONBytes,
		}); err != nil {
			return nil, chooseErrMsg("could not log txn splits",
				"could not log transaction splits"), nil
		}
	}
	return &newTxn, "", nil
}
