package api

import (
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/YouWantToPinch/pincher-api/internal/database"
)

func (cfg *APIConfig) handleAssignAmountToCategory(w http.ResponseWriter, r *http.Request) {
	type rqSchema struct {
		Amount int64 `json:"amount"`
	}

	rqPayload, err := decodePayload[rqSchema](r)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "", err)
		return
	}

	if rqPayload.Amount == 0 {
		respondWithError(w, http.StatusBadRequest, "could not assign non-zero amount", nil)
		return
	}

	parsedMonth, err := parseDateFromPath("month_id", r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "", err)
		return
	}

	parsedCategoryID, err := parseUUIDFromPath("category_id", r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "", err)
		return
	}

	dbAssignment, err := cfg.db.AssignAmountToCategory(r.Context(), database.AssignAmountToCategoryParams{
		MonthID:    parsedMonth,
		CategoryID: parsedCategoryID,
		Amount:     rqPayload.Amount,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not assign amount to category for month specified", err)
		return
	}

	type rspSchema struct {
		MonthID    time.Time `json:"month_id"`
		CategoryID uuid.UUID `json:"category_id"`
		Amount     int64     `json:"amount"`
	}

	rspPayload := rspSchema{
		MonthID:    dbAssignment.Month,
		CategoryID: dbAssignment.CategoryID,
		Amount:     dbAssignment.Assigned,
	}

	respondWithJSON(w, http.StatusOK, rspPayload)
}

func (cfg *APIConfig) handleGetMonthReport(w http.ResponseWriter, r *http.Request) {
	parsedMonthID, err := parseDateFromPath("month_id", r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "", err)
		return
	}

	monthReport, err := cfg.db.GetMonthReport(r.Context(), parsedMonthID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not generate report for month specified", err)
		return
	}

	rspPayload := MonthReport{
		Assigned: monthReport.Assigned,
		Activity: monthReport.Activity,
		Balance:  monthReport.Balance,
	}

	respondWithJSON(w, http.StatusOK, rspPayload)
}

func (cfg *APIConfig) handleGetMonthCategories(w http.ResponseWriter, r *http.Request) {
	pathBudgetID := getContextKeyValueAsUUID(r.Context(), "budget_id")

	parsedMonthID, err := parseDateFromPath("month_id", r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "", err)
		return
	}

	dbCategoryReports, err := cfg.db.GetMonthCategoryReports(r.Context(), database.GetMonthCategoryReportsParams{
		MonthID:  parsedMonthID,
		BudgetID: pathBudgetID,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not generate category reports for month specified", err)
		return
	}

	var rspPayload []CategoryReport
	for _, report := range dbCategoryReports {

		newReport := CategoryReport{
			MonthID:  report.Month,
			Name:     report.CategoryName,
			Assigned: report.Assigned,
			Activity: report.Activity,
			Balance:  report.Balance,
		}

		rspPayload = append(rspPayload, newReport)
	}

	respondWithJSON(w, http.StatusOK, rspPayload)
}

func (cfg *APIConfig) handleGetMonthCategoryReport(w http.ResponseWriter, r *http.Request) {
	pathBudgetID := getContextKeyValueAsUUID(r.Context(), "budget_id")

	parsedMonthID, err := parseDateFromPath("month_id", r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "", err)
		return
	}

	pathCategoryID, err := parseUUIDFromPath("category_id", r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "", err)
		return
	}
	dbCategoryReport, err := cfg.db.GetMonthCategoryReport(r.Context(), database.GetMonthCategoryReportParams{
		MonthID:    parsedMonthID,
		CategoryID: pathCategoryID,
		BudgetID:   pathBudgetID,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not generate category report for month specified", err)
		return
	}

	rspPayload := CategoryReport{
		MonthID:  dbCategoryReport.Month,
		Name:     dbCategoryReport.CategoryName,
		Assigned: dbCategoryReport.Assigned,
		Activity: dbCategoryReport.Activity,
		Balance:  dbCategoryReport.Balance,
	}
	respondWithJSON(w, http.StatusOK, rspPayload)
}
