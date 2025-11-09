package api

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/YouWantToPinch/pincher-api/internal/database"
)

func (cfg *apiConfig) endpCreatePayee(w http.ResponseWriter, r *http.Request) {

	type parameters struct {
		Name string `json:"name"`
	}

	var params parameters
	err := decodeParams(r, &params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failure decoding parameters", err)
		return
	}

	pathBudgetID := getContextKeyValue(r.Context(), "budget_id")

	dbPayee, err := cfg.db.CreatePayee(r.Context(), database.CreatePayeeParams{
		BudgetID: pathBudgetID,
		Name:     params.Name,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create payee", err)
		return
	}

	respBody := Payee{
		ID:        dbPayee.ID,
		CreatedAt: dbPayee.CreatedAt,
		UpdatedAt: dbPayee.UpdatedAt,
		BudgetID:  dbPayee.BudgetID,
		Name:      dbPayee.Name,
	}

	respondWithJSON(w, http.StatusCreated, respBody)
}

func (cfg *apiConfig) endpGetPayees(w http.ResponseWriter, r *http.Request) {

	pathBudgetID := getContextKeyValue(r.Context(), "budget_id")
	payees, err := cfg.db.GetBudgetPayees(r.Context(), pathBudgetID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get budget payees", err)
		return
	}

	var respBody []Payee
	for _, payee := range payees {
		addPayee := Payee{
			ID:        payee.ID,
			CreatedAt: payee.CreatedAt,
			UpdatedAt: payee.UpdatedAt,
			BudgetID:  payee.BudgetID,
			Name:      payee.Name,
		}
		respBody = append(respBody, addPayee)
	}

	respondWithJSON(w, http.StatusOK, respBody)
}

func (cfg *apiConfig) endpGetPayee(w http.ResponseWriter, r *http.Request) {

	idString := r.PathValue("payee_id")
	pathPayeeID, err := uuid.Parse(idString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid id", err)
		return
	}

	dbPayee, err := cfg.db.GetPayeeByID(r.Context(), pathPayeeID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Couldn't find payee with specified id", err)
		return
	}

	respBody := Payee{
		ID:        dbPayee.ID,
		CreatedAt: dbPayee.CreatedAt,
		UpdatedAt: dbPayee.UpdatedAt,
		BudgetID:  dbPayee.BudgetID,
		Name:      dbPayee.Name,
	}

	respondWithJSON(w, http.StatusCreated, respBody)
}

func (cfg *apiConfig) endpDeletePayee(w http.ResponseWriter, r *http.Request) {

	idString := r.PathValue("payee_id")
	pathPayeeID, err := uuid.Parse(idString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid id", err)
		return
	}

	dbPayee, err := cfg.db.GetPayeeByID(r.Context(), pathPayeeID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Couldn't find payee with specified id", err)
		return
	}

	pathBudgetID := getContextKeyValue(r.Context(), "budget_id")
	if pathBudgetID != dbPayee.BudgetID {
		respondWithError(w, http.StatusForbidden, "401 Unauthorized", nil)
		return
	}

	err = cfg.db.DeletePayee(r.Context(), pathPayeeID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to delete payee from budget", err)
		return
	}

	respondWithText(w, http.StatusNoContent, "The payee was deleted.")
}
