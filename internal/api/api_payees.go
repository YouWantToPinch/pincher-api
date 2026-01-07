package api

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/YouWantToPinch/pincher-api/internal/database"
)

func (cfg *APIConfig) endpCreatePayee(w http.ResponseWriter, r *http.Request) {
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

func (cfg *APIConfig) endpGetPayees(w http.ResponseWriter, r *http.Request) {
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

func (cfg *APIConfig) endpGetPayee(w http.ResponseWriter, r *http.Request) {
	var pathPayeeID uuid.UUID
	err := parseUUIDFromPath("payee_id", r, &pathPayeeID)
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

func (cfg *APIConfig) endpUpdatePayee(w http.ResponseWriter, r *http.Request) {
	var pathPayeeID uuid.UUID
	err := parseUUIDFromPath("payee_id", r, &pathPayeeID)
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

func (cfg *APIConfig) endpDeletePayee(w http.ResponseWriter, r *http.Request) {
	var pathPayeeID uuid.UUID
	err := parseUUIDFromPath("payee_id", r, &pathPayeeID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "", err)
		return
	}

	dbPayee, err := cfg.db.GetPayeeByID(r.Context(), pathPayeeID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "could not get payee", err)
		return
	}

	pathBudgetID := getContextKeyValueAsUUID(r.Context(), "budget_id")
	if pathBudgetID != dbPayee.BudgetID {
		respondWithCode(w, http.StatusForbidden)
		return
	}

	type rqSchema struct {
		NewPayeeName string `json:"new_payee_name"`
	}

	rqPayload, err := decodePayload[rqSchema](r)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "", err)
		return
	}

	payeeInUse, err := cfg.db.IsPayeeInUse(r.Context(), pathPayeeID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not determine whether payee in use", err)
	}

	if payeeInUse {
		if rqPayload.NewPayeeName == "" {
			respondWithError(w, http.StatusBadRequest, "payee ID not provided", nil)
			return
		}
		PayeeID, err := lookupResourceIDByName(r.Context(),
			database.GetBudgetPayeeIDByNameParams{
				PayeeName: rqPayload.NewPayeeName,
				BudgetID:  pathBudgetID,
			}, cfg.db.GetBudgetPayeeIDByName)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "could not get payee id", err)
			return
		}

		err = cfg.db.ReassignTransactions(r.Context(), database.ReassignTransactionsParams{
			OldPayeeID: pathPayeeID,
			NewPayeeID: *PayeeID,
		})
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "could not reassign payee for transactions", err)
			return
		}
	}

	err = cfg.db.DeletePayee(r.Context(), pathPayeeID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not delete payee", err)
		return
	}

	respondWithCode(w, http.StatusNoContent)
}
