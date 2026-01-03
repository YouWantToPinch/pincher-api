package api

import (
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/YouWantToPinch/pincher-api/internal/database"
)

func (cfg *APIConfig) endpAssignAmountToCategory(w http.ResponseWriter, r *http.Request) {
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

	var parsedMonth time.Time
	err = parseDateFromPath("month_id", r, &parsedMonth)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "", err)
		return
	}

	var parsedCategoryID uuid.UUID
	err = parseUUIDFromPath("category_id", r, &parsedCategoryID)
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

	type Assignment struct {
		MonthID    time.Time `json:"month_id"`
		CategoryID uuid.UUID `json:"category_id"`
		Amount     int64     `json:"amount"`
	}

	rspPayload := Assignment{
		MonthID:    dbAssignment.Month,
		CategoryID: dbAssignment.CategoryID,
		Amount:     dbAssignment.Assigned,
	}

	respondWithJSON(w, http.StatusCreated, rspPayload)
}

func (cfg *APIConfig) endpGetMonthReport(w http.ResponseWriter, r *http.Request) {
	var parsedMonth time.Time
	err := parseDateFromPath("month_id", r, &parsedMonth)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "", err)
		return
	}

	monthReport, err := cfg.db.GetMonthReport(r.Context(), parsedMonth)
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

func (cfg *APIConfig) endpGetMonthCategories(w http.ResponseWriter, r *http.Request) {
	var parsedMonth time.Time
	err := parseDateFromPath("month_id", r, &parsedMonth)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "", err)
		return
	}

	dbCategoryReports, err := cfg.db.GetMonthCategoryReports(r.Context(), parsedMonth)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not generate category reports for month specified", err)
		return
	}

	var rspPayload []CategoryReport
	for _, report := range dbCategoryReports {

		newReport := CategoryReport{
			MonthID:    report.Month,
			CategoryID: report.CategoryID,
			Name:       report.CategoryName,
			Assigned:   report.Assigned,
			Activity:   report.Activity,
			Balance:    report.Balance,
		}

		rspPayload = append(rspPayload, newReport)
	}

	respondWithJSON(w, http.StatusOK, rspPayload)
}

func (cfg *APIConfig) endpGetMonthCategoryReport(w http.ResponseWriter, r *http.Request) {
	var parsedMonth time.Time
	err := parseDateFromPath("month_id", r, &parsedMonth)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "", err)
		return
	}

	var pathCategoryID uuid.UUID
	err = parseUUIDFromPath("category_id", r, &pathCategoryID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "", err)
		return
	}
	dbCategoryReport, err := cfg.db.GetMonthCategoryReport(r.Context(), database.GetMonthCategoryReportParams{
		Month:      parsedMonth,
		CategoryID: pathCategoryID,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not generate category report for month specified", err)
		return
	}

	rspPayload := CategoryReport{
		MonthID:    dbCategoryReport.Month,
		CategoryID: dbCategoryReport.CategoryID,
		Name:       dbCategoryReport.CategoryName,
		Assigned:   dbCategoryReport.Assigned,
		Activity:   dbCategoryReport.Activity,
		Balance:    dbCategoryReport.Balance,
	}
	respondWithJSON(w, http.StatusOK, rspPayload)
}
