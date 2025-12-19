package api

import (
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/YouWantToPinch/pincher-api/internal/database"
)

func (cfg *APIConfig) endpCreateBudget(w http.ResponseWriter, r *http.Request) {
	validatedUserID := getContextKeyValue(r.Context(), "user_id")
	slog.Debug("user_id is " + validatedUserID.String())

	type rqSchema struct {
		Meta
	}

	rqPayload, err := decodePayload[rqSchema](r)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failure decoding parameters", err)
		return
	}

	if rqPayload.Name == "" {
		respondWithError(w, http.StatusBadRequest, "Name not provided", nil)
		return
	}

	dbBudget, err := cfg.db.CreateBudget(r.Context(), database.CreateBudgetParams{
		AdminID: validatedUserID,
		Name:    rqPayload.Name,
		Notes:   rqPayload.Notes,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create budget", err)
		return
	}

	_, err = cfg.db.AssignBudgetMemberWithRole(r.Context(), database.AssignBudgetMemberWithRoleParams{
		BudgetID:   dbBudget.ID,
		UserID:     validatedUserID,
		MemberRole: "ADMIN",
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to assign ADMIN to new budget", err)
		slog.Info("Attempting deletion of new budget, as no admin could be assigned.")
		err := cfg.db.DeleteBudget(r.Context(), dbBudget.ID)
		if err != nil {
			slog.Warn("Attempted deletion of newly initialized budget, but FAILED.")
			return
		}
		slog.Info("Initialized budget was deleted successfully.")
		return
	}

	rspPayload := Budget{
		ID:        dbBudget.ID,
		CreatedAt: dbBudget.CreatedAt,
		UpdatedAt: dbBudget.UpdatedAt,
		AdminID:   dbBudget.AdminID,
	}

	respondWithJSON(w, http.StatusCreated, rspPayload)
}

func (cfg *APIConfig) endpGetBudget(w http.ResponseWriter, r *http.Request) {
	pathBudgetID := getContextKeyValue(r.Context(), "budget_id")

	dbBudget, err := cfg.db.GetBudgetByID(r.Context(), pathBudgetID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "No budget membership found for user", err)
		return
	}

	rspPayload := Budget{
		ID:        dbBudget.ID,
		CreatedAt: dbBudget.CreatedAt,
		UpdatedAt: dbBudget.UpdatedAt,
		AdminID:   dbBudget.AdminID,
		Meta: Meta{
			Name:  dbBudget.Name,
			Notes: dbBudget.Notes,
		},
	}

	respondWithJSON(w, http.StatusOK, rspPayload)
}

func (cfg *APIConfig) endpGetUserBudgets(w http.ResponseWriter, r *http.Request) {
	validatedUserID := getContextKeyValue(r.Context(), "user_id")

	roleFilters := r.URL.Query()["role"]

	var dbBudgets []database.Budget
	var err error
	if len(roleFilters) == 0 {
		dbBudgets, err = cfg.db.GetUserBudgets(r.Context(), database.GetUserBudgetsParams{
			UserID: validatedUserID,
		})
		if err != nil {
			respondWithError(w, http.StatusNotFound, "No budget memberships found for user", err)
			return
		}
	} else {
		dbBudgets, err = cfg.db.GetUserBudgets(r.Context(), database.GetUserBudgetsParams{
			UserID: validatedUserID,
			Roles:  roleFilters,
		})
		if err != nil {
			respondWithError(w, http.StatusNotFound, "No budgets found with specified membership role", err)
			return
		}
	}

	var budgets []Budget
	for _, dbBudget := range dbBudgets {
		addBudget := Budget{
			ID:        dbBudget.ID,
			CreatedAt: dbBudget.CreatedAt,
			UpdatedAt: dbBudget.UpdatedAt,
			AdminID:   dbBudget.AdminID,
			Meta: Meta{
				Name:  dbBudget.Name,
				Notes: dbBudget.Notes,
			},
		}
		budgets = append(budgets, addBudget)
	}

	type rspSchema struct {
		Budgets []Budget `json:"budgets"`
	}

	rspPayload := rspSchema{
		Budgets: budgets,
	}

	respondWithJSON(w, http.StatusOK, rspPayload)
}

func (cfg *APIConfig) endpGetBudgetCapital(w http.ResponseWriter, r *http.Request) {
	pathBudgetID := getContextKeyValue(r.Context(), "budget_id")
	capitalAmount, err := cfg.db.GetBudgetCapital(r.Context(), pathBudgetID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong", err)
		return
	}

	type rspSchema struct {
		Capital int64 `json:"capital"`
	}

	rspPayload := rspSchema{
		Capital: capitalAmount,
	}

	respondWithJSON(w, http.StatusOK, rspPayload)
}

func (cfg *APIConfig) endpAddBudgetMemberWithRole(w http.ResponseWriter, r *http.Request) {
	pathBudgetID := getContextKeyValue(r.Context(), "budget_id")

	type rqSchema struct {
		UserID     string `json:"user_id"`
		MemberRole string `json:"member_role"`
	}

	rqPayload, err := decodePayload[rqSchema](r)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failure decoding parameters", err)
		return
	}

	parsedUserID, err := uuid.Parse(rqPayload.UserID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong", err)
		slog.Error("Could not parse user_id")
		return
	}

	newMemberRole, err := BMRFromString(rqPayload.MemberRole)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error(), err)
		return
	}
	if newMemberRole == ADMIN {
		respondWithError(w, http.StatusBadRequest, "ADMIN assignment requested; DENIED. Only one allowed, set only by creating a new budget.", nil)
		return
	}

	dbBudgetMembership, err := cfg.db.AssignBudgetMemberWithRole(r.Context(), database.AssignBudgetMemberWithRoleParams{
		BudgetID:   pathBudgetID,
		UserID:     parsedUserID,
		MemberRole: newMemberRole.String(),
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to assign new member to budget", err)
		return
	}

	rspPayload := BudgetMembership{
		BudgetID:   dbBudgetMembership.BudgetID,
		UserID:     dbBudgetMembership.UserID,
		MemberRole: newMemberRole,
	}

	respondWithJSON(w, http.StatusCreated, rspPayload)
}

func (cfg *APIConfig) endpRemoveBudgetMember(w http.ResponseWriter, r *http.Request) {
	pathBudgetID := getContextKeyValue(r.Context(), "budget_id")

	var pathUserID uuid.UUID
	err := parseUUIDFromPath("user_id", r, &pathUserID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong", err)
		slog.Error("Could not parse user_id")
		return
	}

	err = cfg.db.RevokeBudgetMembership(r.Context(), database.RevokeBudgetMembershipParams{
		BudgetID: pathBudgetID,
		UserID:   pathUserID,
	})
	if err != nil {
		respondWithText(w, http.StatusNotFound, "Failed to find membership to revoke")
	}

	respondWithText(w, http.StatusNoContent, "Revoked membership successfully")
}

func (cfg *APIConfig) endpUpdateBudget(w http.ResponseWriter, r *http.Request) {
	pathBudgetID := getContextKeyValue(r.Context(), "budget_id")

	type rqSchema struct {
		Meta
	}

	rqPayload, err := decodePayload[rqSchema](r)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failure decoding parameters", err)
		return
	}

	_, err = cfg.db.UpdateBudget(r.Context(), database.UpdateBudgetParams{
		ID:    pathBudgetID,
		Name:  rqPayload.Name,
		Notes: rqPayload.Notes,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update budget", err)
		return
	}

	respondWithText(w, http.StatusNoContent, "Budget '"+rqPayload.Name+"' updated successfully!")
}

func (cfg *APIConfig) endpDeleteBudget(w http.ResponseWriter, r *http.Request) {
	pathBudgetID := getContextKeyValue(r.Context(), "budget_id")

	err := cfg.db.DeleteBudget(r.Context(), pathBudgetID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "404 Not Found", err)
		return
	}

	respondWithText(w, http.StatusNoContent, "Deleted budget")
}
