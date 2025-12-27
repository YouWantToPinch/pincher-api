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
		respondWithError(w, http.StatusInternalServerError, "failure decoding request payload: ", err)
		return
	}

	if rqPayload.Name == "" {
		respondWithError(w, http.StatusBadRequest, "name not provided", nil)
		return
	}

	pathBudgetID := getContextKeyValue(r.Context(), "budget_id")

	dbPayee, err := cfg.db.CreatePayee(r.Context(), database.CreatePayeeParams{
		BudgetID: pathBudgetID,
		Name:     rqPayload.Name,
		Notes:    rqPayload.Notes,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not create payee: ", err)
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
	pathBudgetID := getContextKeyValue(r.Context(), "budget_id")
	dbPayees, err := cfg.db.GetBudgetPayees(r.Context(), pathBudgetID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not get budget payees: ", err)
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
		respondWithError(w, http.StatusBadRequest, "failure parsing UUID: ", err)
		return
	}

	dbPayee, err := cfg.db.GetPayeeByID(r.Context(), pathPayeeID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "could not find payee: ", err)
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

func (cfg *APIConfig) endpUpdatePayee(w http.ResponseWriter, r *http.Request) {
	var pathPayeeID uuid.UUID
	err := parseUUIDFromPath("payee_id", r, &pathPayeeID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "failure parsing UUID: ", err)
		return
	}

	type rqSchema struct {
		Meta
	}

	rqPayload, err := decodePayload[rqSchema](r)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failure decoding request payload: ", err)
		return
	}

	_, err = cfg.db.UpdatePayee(r.Context(), database.UpdatePayeeParams{
		ID:    pathPayeeID,
		Name:  rqPayload.Name,
		Notes: rqPayload.Notes,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to update payee", err)
		return
	}

	respondWithText(w, http.StatusOK, "Payee '"+rqPayload.Name+"' updated successfully")
}

func (cfg *APIConfig) endpDeletePayee(w http.ResponseWriter, r *http.Request) {
	var pathPayeeID uuid.UUID
	err := parseUUIDFromPath("payee_id", r, &pathPayeeID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid parameter value", err)
		return
	}

	dbPayee, err := cfg.db.GetPayeeByID(r.Context(), pathPayeeID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "could not find payee: ", err)
		return
	}

	pathBudgetID := getContextKeyValue(r.Context(), "budget_id")
	if pathBudgetID != dbPayee.BudgetID {
		respondWithError(w, http.StatusForbidden, "401 Unauthorized", nil)
		return
	}

	type rqSchema struct {
		NewPayeeID string `json:"new_payee_id"`
	}

	rqPayload, err := decodePayload[rqSchema](r)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failure decoding request payload: ", err)
		return
	}

	payeeInUse, err := cfg.db.IsPayeeInUse(r.Context(), pathPayeeID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not determine whether payee in use: ", err)
	}

	if payeeInUse {
		if rqPayload.NewPayeeID == "" {
			respondWithError(w, http.StatusBadRequest, "payee_id not provided", nil)
			return
		}
		parsedNewPayeeID, err := uuid.Parse(rqPayload.NewPayeeID)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "failure parsing payee ID: ", err)
			return
		}
		err = cfg.db.ReassignTransactions(r.Context(), database.ReassignTransactionsParams{
			OldPayeeID: pathPayeeID,
			NewPayeeID: parsedNewPayeeID,
		})
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "could not reassign payee for transactions", err)
			return
		}
	}

	err = cfg.db.DeletePayee(r.Context(), pathPayeeID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to delete payee from budget", err)
		return
	}

	respondWithText(w, http.StatusOK, "Payee deleted successfully")
}
