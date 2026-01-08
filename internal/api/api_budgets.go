package api

import (
	"log/slog"
	"net/http"

	"github.com/YouWantToPinch/pincher-api/internal/database"
)

func (cfg *APIConfig) handleCreateBudget(w http.ResponseWriter, r *http.Request) {
	validatedUserID := getContextKeyValueAsUUID(r.Context(), "user_id")

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

	dbBudget, err := cfg.db.CreateBudget(r.Context(), database.CreateBudgetParams{
		AdminID: validatedUserID,
		Name:    rqPayload.Name,
		Notes:   rqPayload.Notes,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not create budget", err)
		return
	}

	_, err = cfg.db.AssignBudgetMemberWithRole(r.Context(), database.AssignBudgetMemberWithRoleParams{
		BudgetID:   dbBudget.ID,
		UserID:     validatedUserID,
		MemberRole: "ADMIN",
	})
	if err != nil {

		// TODO: use db transaction w/ rollback for atomicity here, as has been done with transactions!

		respondWithError(w, http.StatusInternalServerError, "could not assign ADMIN to new budget", err)
		slog.Debug("Attempting deletion of new budget, as no admin could be assigned.")
		err := cfg.db.DeleteBudget(r.Context(), dbBudget.ID)
		if err != nil {
			slog.Warn("Attempted deletion of newly initialized budget, but FAILED.")
		} else {
			slog.Info("Initialized budget was deleted successfully.")
		}
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

func (cfg *APIConfig) handleGetBudget(w http.ResponseWriter, r *http.Request) {
	pathBudgetID := getContextKeyValueAsUUID(r.Context(), "budget_id")

	dbBudget, err := cfg.db.GetBudgetByID(r.Context(), pathBudgetID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "could not get budget", err)
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

func (cfg *APIConfig) handleGetUserBudgets(w http.ResponseWriter, r *http.Request) {
	validatedUserID := getContextKeyValueAsUUID(r.Context(), "user_id")

	roleFilters := r.URL.Query()["role"]

	var dbBudgets []database.Budget
	var err error
	if len(roleFilters) == 0 {
		dbBudgets, err = cfg.db.GetUserBudgets(r.Context(), database.GetUserBudgetsParams{
			UserID: validatedUserID,
		})
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "could not retrieve user budgets", err)
			return
		}
	} else {
		dbBudgets, err = cfg.db.GetUserBudgets(r.Context(), database.GetUserBudgetsParams{
			UserID: validatedUserID,
			Roles:  roleFilters,
		})
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "could not retrieve user budgets", err)
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

func (cfg *APIConfig) handleGetBudgetCapital(w http.ResponseWriter, r *http.Request) {
	pathBudgetID := getContextKeyValueAsUUID(r.Context(), "budget_id")
	capitalAmount, err := cfg.db.GetBudgetCapital(r.Context(), pathBudgetID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not calculate budget capital", err)
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

func (cfg *APIConfig) handleAddBudgetMemberWithRole(w http.ResponseWriter, r *http.Request) {
	pathBudgetID := getContextKeyValueAsUUID(r.Context(), "budget_id")

	type rqSchema struct {
		UserName   string `json:"username"`
		MemberRole string `json:"member_role"`
	}

	rqPayload, err := decodePayload[rqSchema](r)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "", err)
		return
	}

	user, err := cfg.db.GetUserByUsername(r.Context(), rqPayload.UserName)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "user does not exist", err)
		return
	}

	newMemberRole, err := BMRFromString(rqPayload.MemberRole)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "failure interpreting role", err)
		return
	}
	if newMemberRole == ADMIN {
		respondWithError(w, http.StatusBadRequest, "could not add admin to budget (only one is permitted)", nil)
		return
	}

	dbBudgetMembership, err := cfg.db.AssignBudgetMemberWithRole(r.Context(), database.AssignBudgetMemberWithRoleParams{
		BudgetID:   pathBudgetID,
		UserID:     user.ID,
		MemberRole: newMemberRole.String(),
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not assign new member to budget", err)
		return
	}

	rspPayload := BudgetMembership{
		BudgetID:   dbBudgetMembership.BudgetID,
		UserID:     dbBudgetMembership.UserID,
		MemberRole: newMemberRole,
	}

	respondWithJSON(w, http.StatusCreated, rspPayload)
}

func (cfg *APIConfig) handleRemoveBudgetMember(w http.ResponseWriter, r *http.Request) {
	pathBudgetID := getContextKeyValueAsUUID(r.Context(), "budget_id")

	pathUserID, err := parseUUIDFromPath("user_id", r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "", err)
		return
	}

	err = cfg.db.RevokeBudgetMembership(r.Context(), database.RevokeBudgetMembershipParams{
		BudgetID: pathBudgetID,
		UserID:   pathUserID,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not revoke membership", err)
	}

	respondWithCode(w, http.StatusNoContent)
}

func (cfg *APIConfig) handleUpdateBudget(w http.ResponseWriter, r *http.Request) {
	pathBudgetID := getContextKeyValueAsUUID(r.Context(), "budget_id")

	type rqSchema struct {
		Meta
	}

	rqPayload, err := decodePayload[rqSchema](r)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "", err)
		return
	}

	_, err = cfg.db.UpdateBudget(r.Context(), database.UpdateBudgetParams{
		ID:    pathBudgetID,
		Name:  rqPayload.Name,
		Notes: rqPayload.Notes,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not update budget", err)
		return
	}

	respondWithCode(w, http.StatusNoContent)
}

func (cfg *APIConfig) handleDeleteBudget(w http.ResponseWriter, r *http.Request) {
	pathBudgetID := getContextKeyValueAsUUID(r.Context(), "budget_id")

	err := cfg.db.DeleteBudget(r.Context(), pathBudgetID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "could not delete budget", err)
		return
	}

	respondWithCode(w, http.StatusNoContent)
}
