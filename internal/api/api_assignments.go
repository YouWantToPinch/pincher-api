package api

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/YouWantToPinch/pincher-api/internal/database"
)

func (cfg *APIConfig) handleAssignAmountToCategory(w http.ResponseWriter, r *http.Request) {
	type rqSchema struct {
		Amount       int64  `json:"amount"`
		ToCategory   string `json:"to_category"`
		FromCategory string `json:"from_category"`
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

	pathBudgetID := getContextKeyValueAsUUID(r.Context(), "budget_id")

	usingDBTxn := rqPayload.FromCategory != ""

	q := cfg.db
	var tx pgx.Tx

	if usingDBTxn {
		// USE DB TRANSACTION
		{
			tx, err = cfg.Pool.Begin(r.Context())
			if err != nil {
				respondWithError(w, http.StatusInternalServerError, "", err)
				return
			}
			defer tx.Rollback(r.Context())

			q = cfg.db.WithTx(tx)

		}
	}

	assignToCat := func(categoryName string, amount int64) (database.Assignment, error) {
		parsedCategoryID := uuid.Nil
		if rqPayload.ToCategory != "" {
			parsedCategoryID, err = lookupResourceIDByName(r.Context(),
				database.GetBudgetCategoryIDByNameParams{
					CategoryName: categoryName,
					BudgetID:     pathBudgetID,
				}, q.GetBudgetCategoryIDByName)
			if err != nil {
				respondWithError(w, http.StatusBadRequest, "could not get category by given name", err)
				return database.Assignment{}, err
			}
		}

		dbAssignment, err := q.AssignAmountToCategory(r.Context(), database.AssignAmountToCategoryParams{
			MonthID:    parsedMonth,
			CategoryID: parsedCategoryID,
			Amount:     amount,
		})
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "could not assign amount to category for month specified", err)
			return database.Assignment{}, err
		}
		return dbAssignment, err
	}
	dbAssignment, err := assignToCat(rqPayload.ToCategory, rqPayload.Amount)
	if err != nil {
		return
	}
	if usingDBTxn {
		_, err = assignToCat(rqPayload.FromCategory, rqPayload.Amount*-1)
		if err != nil {
			return
		}
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
	if usingDBTxn {
		if err := tx.Commit(r.Context()); err != nil {
			respondWithError(w, http.StatusInternalServerError, "", err)
			return
		}
	}

	respondWithJSON(w, http.StatusOK, rspPayload)
}

func (cfg *APIConfig) handleGetMonthReport(w http.ResponseWriter, r *http.Request) {
	parsedMonthID, err := parseDateFromPath("month_id", r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "", err)
		return
	}

	pathBudgetID := getContextKeyValueAsUUID(r.Context(), "budget_id")

	monthReport, err := cfg.db.GetMonthReport(r.Context(), database.GetMonthReportParams{
		MonthID:  parsedMonthID,
		BudgetID: pathBudgetID,
	})
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

	var categoryReports []CategoryReport
	for _, report := range dbCategoryReports {
		categoryReports = append(categoryReports, CategoryReport{
			MonthID:  report.Month,
			Name:     report.CategoryName,
			Assigned: report.Assigned,
			Activity: report.Activity,
			Balance:  report.Balance,
		})
	}

	type rspSchema struct {
		CategoryReports []CategoryReport `json:"category_reports"`
	}

	rspPayload := rspSchema{
		CategoryReports: categoryReports,
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
