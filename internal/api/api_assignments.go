package api

import (
	"net/http"
	"time"

	db "github.com/YouWantToPinch/pincher-api/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

	assignToCat := func(categoryName string, amount int64) (db.Assignment, error) {
		parsedCategoryID := uuid.Nil
		if rqPayload.ToCategory != "" {
			parsedCategoryID, err = lookupResourceIDByName(r.Context(),
				db.GetBudgetCategoryIDByNameParams{
					CategoryName: categoryName,
					BudgetID:     pathBudgetID,
				}, q.GetBudgetCategoryIDByName)
			if err != nil {
				respondWithError(w, http.StatusBadRequest, "could not get category by given name", err)
				return db.Assignment{}, err
			}
		}

		dbAssignment, err := q.AssignAmountToCategory(r.Context(), db.AssignAmountToCategoryParams{
			MonthID:    parsedMonth,
			CategoryID: parsedCategoryID,
			Amount:     amount,
		})
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "could not assign amount to category for month specified", err)
			return db.Assignment{}, err
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

	monthReport, err := cfg.db.GetMonthReport(r.Context(), db.GetMonthReportParams{
		MonthID:  parsedMonthID,
		BudgetID: pathBudgetID,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not generate report for month specified", err)
		return
	}

	rspPayload := MonthReport{
		MonthID:  monthReport.Month,
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

	dbCategoryReports, err := cfg.db.GetMonthCategoryReports(r.Context(), db.GetMonthCategoryReportsParams{
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
			MonthID:    report.Month,
			CategoryID: report.CategoryID,
			GroupID:    report.GroupID,
			Name:       report.CategoryName,
			Assigned:   report.Assigned,
			Activity:   report.Activity,
			Balance:    report.Balance,
		})
	}

	type rspSchema struct {
		CategoryReports []CategoryReport `json:"data"`
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
	dbCategoryReport, err := cfg.db.GetMonthCategoryReport(r.Context(), db.GetMonthCategoryReportParams{
		MonthID:    parsedMonthID,
		CategoryID: pathCategoryID,
		BudgetID:   pathBudgetID,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not generate category report for month specified", err)
		return
	}

	rspPayload := CategoryReport{
		MonthID:    dbCategoryReport.Month,
		CategoryID: dbCategoryReport.CategoryID,
		GroupID:    dbCategoryReport.GroupID,
		Name:       dbCategoryReport.CategoryName,
		Assigned:   dbCategoryReport.Assigned,
		Activity:   dbCategoryReport.Activity,
		Balance:    dbCategoryReport.Balance,
	}
	respondWithJSON(w, http.StatusOK, rspPayload)
}

func (cfg *APIConfig) handleGetMonthGroups(w http.ResponseWriter, r *http.Request) {
	pathBudgetID := getContextKeyValueAsUUID(r.Context(), "budget_id")

	parsedMonthID, err := parseDateFromPath("month_id", r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "", err)
		return
	}

	dbGroupReports, err := cfg.db.GetMonthGroupReports(r.Context(), db.GetMonthGroupReportsParams{
		MonthID:  parsedMonthID,
		BudgetID: pathBudgetID,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not generate group reports for month specified", err)
		return
	}

	var groupReports []GroupReport
	for _, report := range dbGroupReports {
		groupReports = append(groupReports, GroupReport{
			MonthID:  report.Month,
			Name:     report.GroupName,
			GroupID:  report.GroupID,
			Assigned: report.Assigned,
			Activity: report.Activity,
			Balance:  report.Balance,
		})
	}

	type rspSchema struct {
		GroupReports []GroupReport `json:"data"`
	}

	rspPayload := rspSchema{
		GroupReports: groupReports,
	}

	respondWithJSON(w, http.StatusOK, rspPayload)
}

func (cfg *APIConfig) handleGetMonthGroupReport(w http.ResponseWriter, r *http.Request) {
	pathBudgetID := getContextKeyValueAsUUID(r.Context(), "budget_id")

	parsedMonthID, err := parseDateFromPath("month_id", r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "", err)
		return
	}

	pathGroupID, err := parseUUIDFromPath("group_id", r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "", err)
		return
	}
	dbGroupReport, err := cfg.db.GetMonthGroupReport(r.Context(), db.GetMonthGroupReportParams{
		MonthID:  parsedMonthID,
		GroupID:  pathGroupID,
		BudgetID: pathBudgetID,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not generate group report for month specified", err)
		return
	}

	rspPayload := GroupReport{
		MonthID:  dbGroupReport.Month,
		GroupID:  dbGroupReport.GroupID,
		Name:     dbGroupReport.GroupName,
		Assigned: dbGroupReport.Assigned,
		Activity: dbGroupReport.Activity,
		Balance:  dbGroupReport.Balance,
	}
	respondWithJSON(w, http.StatusOK, rspPayload)
}
