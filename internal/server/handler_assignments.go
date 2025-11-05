package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/YouWantToPinch/pincher-api/internal/database"
)

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

	var parsedCategoryID uuid.UUID
	err = parseUUIDFromPath("category_id", r, &parsedCategoryID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid parameter value", err)
		return
	}

	dbAssignment, err := cfg.db.AssignAmountToCategory(r.Context(), database.AssignAmountToCategoryParams{
		MonthID:    parsedMonth,
		CategoryID: parsedCategoryID,
		Amount:     params.Amount,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't assign amount to category for month specified", err)
		return
	}

	respondWithJSON(w, http.StatusCreated, dbAssignment)
}

func (cfg *apiConfig) endpGetMonthReport(w http.ResponseWriter, r *http.Request) {
	var parsedMonth time.Time
	err := parseDateFromPath("month_id", r, &parsedMonth)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid parameter value input for month", err)
		return
	}

	monthReport, err := cfg.db.GetMonthReport(r.Context(), parsedMonth)

	respondWithJSON(w, http.StatusOK, monthReport)
}

func (cfg *apiConfig) endpGetMonthCategories(w http.ResponseWriter, r *http.Request) {

	var parsedMonth time.Time
	err := parseDateFromPath("month_id", r, &parsedMonth)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid parameter value input for month", err)
		return
	}

	dbCategoryReports, err := cfg.db.GetMonthCategoryReports(r.Context(), parsedMonth)

	var respBody []CategoryReport
	for _, report := range dbCategoryReports {

		newReport := CategoryReport{
			MonthID:    report.Month,
			CategoryID: report.CategoryID,
			Name:       report.CategoryName,
			Assigned:   report.Assigned,
			Activity:   report.Activity,
			Balance:    report.Balance,
		}

		respBody = append(respBody, newReport)
	}

	respondWithJSON(w, http.StatusOK, respBody)
}

func (cfg *apiConfig) endpGetMonthCategoryReport(w http.ResponseWriter, r *http.Request) {

	var parsedMonth time.Time
	err := parseDateFromPath("month_id", r, &parsedMonth)
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
	dbCategoryReport, err := cfg.db.GetMonthCategoryReport(r.Context(), database.GetMonthCategoryReportParams{
		Month:      parsedMonth,
		CategoryID: pathCategoryID,
	})

	respBody := CategoryReport{
		MonthID:    dbCategoryReport.Month,
		CategoryID: dbCategoryReport.CategoryID,
		Name:       dbCategoryReport.CategoryName,
		Assigned:   dbCategoryReport.Assigned,
		Activity:   dbCategoryReport.Activity,
		Balance:    dbCategoryReport.Balance,
	}
	respondWithJSON(w, http.StatusOK, respBody)
}
