package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/YouWantToPinch/pincher-api/internal/database"
)

func parseDateFromPath(pathParam string, r *http.Request, parse *time.Time) error {
	const timeLayout = time.RFC3339
	dateString := r.PathValue(pathParam)
	if dateString != "" {
		parsedDate, err := time.Parse(timeLayout, dateString)
		if err != nil {
			return fmt.Errorf("Path value '%s' for provided parameter '%s' could not be parsed as DATE", dateString, pathParam)
		}
		*parse = parsedDate
	} else {
		*parse = time.Time{}
	}
	return nil
}

func (cfg *apiConfig) endpAssignAmountToCategory(w http.ResponseWriter, r *http.Request) {

	type parameters struct {
		Amount int64 `json:"amount"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failure decoding parameters", err)
		return
	}

	if params.Amount == 0 {
		respondWithError(w, http.StatusBadRequest, "Input a non-zero amount to modify the budget assignment for the given month", err)
		return
	}

	var parsedMonth time.Time
	err = parseDateFromPath("month_id", r, &parsedMonth)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid parameter value input for month", err)
		return
	}

	pathCategoryString := r.PathValue("category_id")
	pathCategoryID, err := uuid.Parse(pathCategoryString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid id", err)
		return
	}

	dbAssignment, err := cfg.db.AssignAmountToCategory(r.Context(), database.AssignAmountToCategoryParams{
		MonthID:    parsedMonth,
		CategoryID: pathCategoryID,
		Assigned:   params.Amount,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't assign amount to category for month specified", err)
		return
	}

	respondWithJSON(w, http.StatusCreated, dbAssignment)
}

func (cfg *apiConfig) endpGetMonthReport(w http.ResponseWriter, r *http.Request) {
	// Should respond with the equivalent of 'GetMonthCategory,' but for ALL categories that EXIST.
	// respondWithJSON(w, http.StatusOK, respBody)
}

func (cfg *apiConfig) endpGetMonthCategory(w http.ResponseWriter, r *http.Request) {
	// Should respond with the month_report row for a given cateogory within a given month
	// respondWithJSON(w, http.StatusOK, respBody)
}
