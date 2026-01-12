package api

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/YouWantToPinch/pincher-api/internal/auth"
	"github.com/YouWantToPinch/pincher-api/internal/database"
	"github.com/google/uuid"
)

// ================= MIDDLEWARE ================= //

type ctxKey string

// middlewareAuthenticate authenticates JSON Web Tokens
// before passing off requests to another handler.
func (cfg *APIConfig) middlewareAuthenticate(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString, err := auth.GetBearerToken(r.Header)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "no token found", err)
			return
		}
		validatedUserID, err := auth.ValidateJWT(tokenString, cfg.secret, "HS256")
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "invalid token provided", nil)
			return
		}
		ctxUserID := ctxKey("user_id")
		ctx := context.WithValue(r.Context(), ctxUserID, validatedUserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// middlewareCheckClearance evaluates whether or not a user can perform
// the action that they intend to perform at a persmissions level,
// dependent on their member role within the budget in question.
func (cfg *APIConfig) middlewareCheckClearance(required BudgetMemberRole, next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		validatedUserID := getContextKeyValueAsUUID(r.Context(), "user_id")

		pathBudgetID, err := parseUUIDFromPath("budget_id", r)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "", err)
			return
		}

		callerRole, err := cfg.db.GetBudgetMemberRole(r.Context(), database.GetBudgetMemberRoleParams{
			BudgetID: pathBudgetID,
			UserID:   validatedUserID,
		})
		if err != nil {
			respondWithError(w, http.StatusForbidden, "user not found as member", err)
			return
		}

		callerBudgetMemberRole, err := BMRFromString(callerRole)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "invalid role", err)
			return
		}

		if callerBudgetMemberRole > required {
			respondWithError(w, http.StatusForbidden, "user does not have clearance for action", err)
			return
		}
		ctxBudgetID := ctxKey("budget_id")
		ctx := context.WithValue(r.Context(), ctxBudgetID, pathBudgetID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// middlewareValidateTxn validates transaction request payloads,
// then converts relevant resource names to their corresponding UUIDs where valid,
// in preparation for a database query to log or update the transaction.
func (cfg *APIConfig) middlewareValidateTxn(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pathBudgetID := getContextKeyValueAsUUID(r.Context(), "budget_id")
		rqPayload, err := decodePayload[UpsertTransactionRqSchema](r)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "", err)
			return
		}
		validatedTxn, err := validateTxnInput(&rqPayload)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "", err)
			return
		}
		if validatedTxn.txnType == "NONE" {
			respondWithError(w, http.StatusInternalServerError, "transaction type could not be inferred", nil)
		}

		accountID, err := lookupResourceIDByName(r.Context(),
			database.GetBudgetAccountIDByNameParams{
				AccountName: rqPayload.AccountName,
				BudgetID:    pathBudgetID,
			}, cfg.db.GetBudgetAccountIDByName)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "could not get account id", err)
			return
		}
		validatedTxn.accountID = accountID

		if validatedTxn.isTransfer {
			transferAccountID, err := lookupResourceIDByName(r.Context(),
				database.GetBudgetAccountIDByNameParams{
					AccountName: rqPayload.TransferAccountName,
					BudgetID:    pathBudgetID,
				}, cfg.db.GetBudgetAccountIDByName)
			if err != nil {
				respondWithError(w, http.StatusBadRequest, "could not get transfer account id", err)
				return
			}
			validatedTxn.transferAccountID = transferAccountID
			validatedTxn.payeeID = uuid.Nil
		} else {
			payeeID, err := lookupResourceIDByName(r.Context(),
				database.GetBudgetPayeeIDByNameParams{
					PayeeName: rqPayload.PayeeName,
					BudgetID:  pathBudgetID,
				}, cfg.db.GetBudgetPayeeIDByName)
			if err != nil {
				respondWithError(w, http.StatusBadRequest, "could not get payee id", err)
				return
			}
			validatedTxn.payeeID = payeeID
			validatedTxn.transferAccountID = uuid.Nil
		}

		// convert names to IDs if needed
		for k, v := range rqPayload.Amounts {
			if _, ok := validatedTxn.amounts[k]; !ok {
				// validation already weeded this one out; move on to the next
				continue
			}
			if k == "TRANSFER" || (k == "UNCATEGORIZED" && validatedTxn.txnType == "DEPOSIT") {
				// categories are not relevant
				continue
			}
			categoryID, err := lookupResourceIDByName(r.Context(),
				database.GetBudgetCategoryIDByNameParams{
					CategoryName: k,
					BudgetID:     pathBudgetID,
				}, cfg.db.GetBudgetCategoryIDByName)
			if err != nil {
				errMessage := "could not get category id for transaction split"
				if len(rqPayload.Amounts) > 1 {
					errMessage = "could not get category id for one or more transaction splits"
				}
				respondWithError(w, http.StatusBadRequest, errMessage, err)
				return
			}
			validatedTxn.amounts[categoryID.String()] = v
			delete(validatedTxn.amounts, k)
		}

		ctxValidatedTxn := ctxKey("validated_txn")
		ctx := context.WithValue(r.Context(), ctxValidatedTxn, validatedTxn)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ============== HELPERS =================

func getContextKeyValueAsUUID(ctx context.Context, key string) uuid.UUID {
	contextKeyValue, ok := ctx.Value(ctxKey(key)).(uuid.UUID)
	if !ok {
		slog.Warn("failed to retrieve key from context", slog.String("key", key))
		return uuid.Nil
	}
	return contextKeyValue
}

func getContextKeyValueAsTxn(ctx context.Context, key string) *validatedTxnPayload {
	contextKeyValue, ok := ctx.Value(ctxKey(key)).(*validatedTxnPayload)
	if !ok {
		slog.Warn("failed to retrieve key from context", slog.String("key", key))
		return nil
	}
	return contextKeyValue
}
