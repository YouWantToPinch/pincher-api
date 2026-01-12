package api

import (
	"net/http"

	"github.com/YouWantToPinch/pincher-api/internal/database"
)

func (cfg *APIConfig) handleCreatePayee(w http.ResponseWriter, r *http.Request) {
	type rqSchema struct {
		Meta
	}

	rqPayload, err := decodePayload[rqSchema](r)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "", err)
		return
	}

	if rqPayload.Name == "" {
		respondWithError(w, http.StatusBadRequest, "name not provided", nil)
		return
	}

	pathBudgetID := getContextKeyValueAsUUID(r.Context(), "budget_id")

	dbPayee, err := cfg.db.CreatePayee(r.Context(), database.CreatePayeeParams{
		BudgetID: pathBudgetID,
		Name:     rqPayload.Name,
		Notes:    rqPayload.Notes,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not create payee", err)
		return
	}

	rspPayload := Payee{
		ID:        dbPayee.ID,
		CreatedAt: dbPayee.CreatedAt,
		UpdatedAt: dbPayee.UpdatedAt,
		BudgetID:  dbPayee.BudgetID,
		Meta: Meta{
			Name:  dbPayee.Name,
			Notes: dbPayee.Notes,
		},
	}

	respondWithJSON(w, http.StatusCreated, rspPayload)
}

func (cfg *APIConfig) handleGetPayees(w http.ResponseWriter, r *http.Request) {
	pathBudgetID := getContextKeyValueAsUUID(r.Context(), "budget_id")
	dbPayees, err := cfg.db.GetBudgetPayees(r.Context(), pathBudgetID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not retrieve budget payees", err)
		return
	}

	var payees []Payee
	for _, dbPayee := range dbPayees {
		payees = append(payees, Payee{
			ID:        dbPayee.ID,
			CreatedAt: dbPayee.CreatedAt,
			UpdatedAt: dbPayee.UpdatedAt,
			BudgetID:  dbPayee.BudgetID,
			Meta: Meta{
				Name:  dbPayee.Name,
				Notes: dbPayee.Notes,
			},
		})
	}

	type rspSchema struct {
		Payees []Payee `json:"payees"`
	}

	rspPayload := rspSchema{
		Payees: payees,
	}

	respondWithJSON(w, http.StatusOK, rspPayload)
}

func (cfg *APIConfig) handleGetPayee(w http.ResponseWriter, r *http.Request) {
	pathPayeeID, err := parseUUIDFromPath("payee_id", r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "", err)
		return
	}

	dbPayee, err := cfg.db.GetPayeeByID(r.Context(), pathPayeeID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "could not get payee", err)
		return
	}

	rspPayload := Payee{
		ID:        dbPayee.ID,
		CreatedAt: dbPayee.CreatedAt,
		UpdatedAt: dbPayee.UpdatedAt,
		BudgetID:  dbPayee.BudgetID,
		Meta: Meta{
			Name:  dbPayee.Name,
			Notes: dbPayee.Notes,
		},
	}

	respondWithJSON(w, http.StatusOK, rspPayload)
}

func (cfg *APIConfig) handleUpdatePayee(w http.ResponseWriter, r *http.Request) {
	pathPayeeID, err := parseUUIDFromPath("payee_id", r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "", err)
		return
	}

	type rqSchema struct {
		Meta
	}

	rqPayload, err := decodePayload[rqSchema](r)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "", err)
		return
	}

	_, err = cfg.db.UpdatePayee(r.Context(), database.UpdatePayeeParams{
		ID:    pathPayeeID,
		Name:  rqPayload.Name,
		Notes: rqPayload.Notes,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not update payee", err)
		return
	}

	respondWithCode(w, http.StatusNoContent)
}

func (cfg *APIConfig) handleDeletePayee(w http.ResponseWriter, r *http.Request) {
	pathPayeeID, err := parseUUIDFromPath("payee_id", r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "", err)
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

		dbPayee, err := q.GetPayeeByID(r.Context(), pathPayeeID)
		if err != nil {
			respondWithError(w, http.StatusNotFound, "could not get payee", err)
			return
		}

		pathBudgetID := getContextKeyValueAsUUID(r.Context(), "budget_id")
		if pathBudgetID != dbPayee.BudgetID {
			respondWithCode(w, http.StatusForbidden)
			return
		}

		payeeInUse, err := q.IsPayeeInUse(r.Context(), pathPayeeID)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "could not determine whether payee in use", err)
		}

		if payeeInUse {

			type rqSchema struct {
				NewPayeeName string `json:"new_payee_name"`
			}

			rqPayload, err := decodePayload[rqSchema](r)
			if err != nil {
				respondWithError(w, http.StatusInternalServerError, "", err)
				return
			}

			if rqPayload.NewPayeeName == "" {
				respondWithError(w, http.StatusBadRequest, "replacement payee name not provided", nil)
				return
			}
			PayeeID, err := lookupResourceIDByName(r.Context(),
				database.GetBudgetPayeeIDByNameParams{
					PayeeName: rqPayload.NewPayeeName,
					BudgetID:  pathBudgetID,
				}, q.GetBudgetPayeeIDByName)
			if err != nil {
				respondWithError(w, http.StatusBadRequest, "no payee found for replacement with given name", err)
				return
			}

			err = q.ReassignTransactions(r.Context(), database.ReassignTransactionsParams{
				OldPayeeID: pathPayeeID,
				NewPayeeID: PayeeID,
			})
			if err != nil {
				respondWithError(w, http.StatusInternalServerError, "could not reassign payee for transactions", err)
				return
			}
		}

		err = q.DeletePayee(r.Context(), pathPayeeID)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "could not delete payee", err)
			return
		}
		if err := tx.Commit(r.Context()); err != nil {
			respondWithError(w, http.StatusInternalServerError, "", err)
			return
		}
	}

	respondWithCode(w, http.StatusNoContent)
}
