package main

import (
	"fmt"
	"log"
	"database/sql"
	"net/http"
	"encoding/json"

	"github.com/google/uuid"

	"github.com/YouWantToPinch/pincher-api/internal/database"
)

func(cfg *apiConfig) endpCreateBudget(w http.ResponseWriter, r *http.Request){
	
	log.Println("DEBUG: Gonna try creating a budget")

	validatedUserID := getContextKeyValue(r.Context(), "user_id")
	log.Println(fmt.Sprintf("DEBUG: user_id is %s", validatedUserID))
	
	type parameters struct {
		Name	string	`json:"name"`
		Notes	string	`json:"notes"`
	}

	decoder := json.NewDecoder(r.Body)
    params := parameters{}
    err := decoder.Decode(&params)
    if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failure decoding parameters", err)
		log.Println("DEBUG: Failure decoding parameters")
		return
    }

	log.Println("DEBUG: decoded name & notes")

	dbBudget, err := cfg.db.CreateBudget(r.Context(), database.CreateBudgetParams{
		AdminID: validatedUserID,
		Name: params.Name,
		Notes: sql.NullString{
			String: params.Notes,
			Valid: true,
		},
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create budget", err)
		return
	}

	log.Println("DEBUG: Created budget")

	_, err = cfg.db.AssignBudgetMemberWithRole(r.Context(), database.AssignBudgetMemberWithRoleParams{
		BudgetID:	dbBudget.ID,
		UserID:		validatedUserID,
		MemberRole:	"ADMIN",
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to assign ADMIN to new budget", err)
		log.Println("Attempting deletion of new budget, as no admin could be assigned.")
		err := cfg.db.DeleteBudget(r.Context(), dbBudget.ID)
		if err != nil {
			log.Println("NOTICE: Attempted deletion of newly initialized budget, but FAILED.")
			return
		}
		log.Println("Initialized budget was deleted successfully.")
		return
	}

	respBody := Budget{
		ID: 		dbBudget.ID,
		CreatedAt:	dbBudget.CreatedAt,
		UpdatedAt:	dbBudget.UpdatedAt,
		AdminID:	dbBudget.AdminID,
		Name: 		dbBudget.Name,
		Notes: 		dbBudget.Notes.String,
	}

	respondWithJSON(w, http.StatusCreated, respBody)
	return
}

func(cfg *apiConfig) endpGetBudget(w http.ResponseWriter, r *http.Request){
	
	pathBudgetID := getContextKeyValue(r.Context(), "budget_id")

	dbBudget, err := cfg.db.GetBudgetByID(r.Context(), pathBudgetID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "No budget membership found for user", err)
		return
	}

	respBody := Budget{
		ID:			dbBudget.ID,
		CreatedAt:	dbBudget.CreatedAt,
		UpdatedAt:	dbBudget.UpdatedAt,
		AdminID:	dbBudget.AdminID,
		Name:		dbBudget.Name,
		Notes:		dbBudget.Notes.String,
	}

	respondWithJSON(w, http.StatusOK, respBody)
	return
}

func(cfg *apiConfig) endpGetUserBudgets(w http.ResponseWriter, r *http.Request){

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
			Roles: roleFilters,
		})
		if err != nil {
			respondWithError(w, http.StatusNotFound, "No budgets found with specified membership role", err)
			return
		}
	}

	var respBody []Budget
	for _, dbBudget := range dbBudgets {
		addBudget := Budget{
			ID:			dbBudget.ID,
			CreatedAt:	dbBudget.CreatedAt,
			UpdatedAt:	dbBudget.UpdatedAt,
			AdminID:	dbBudget.AdminID,
			Name:		dbBudget.Name,
			Notes:		dbBudget.Notes.String,
		}
		respBody = append(respBody, addBudget)
	}

	respondWithJSON(w, http.StatusOK, respBody)
	return
}

func(cfg *apiConfig) endpAddBudgetMemberWithRole(w http.ResponseWriter, r *http.Request){
	
	pathBudgetID := getContextKeyValue(r.Context(), "budget_id")
	
	type parameters struct {
		UserID		string	`json:"user_id"`
		MemberRole	string	`json:"member_role"`
	}

    decoder := json.NewDecoder(r.Body)
    params := parameters{}
    err := decoder.Decode(&params)
    if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failure decoding parameters", err)
		return
    }

	parsedUserID, err := uuid.Parse(params.UserID)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong", err)
		log.Println("Could not parse user_id")
		return
	}
	
	newMemberRole, err := BMRFromString(params.MemberRole)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error(), err)
		return
	}
	if newMemberRole == ADMIN {
		respondWithError(w, http.StatusBadRequest, "ADMIN assignment requested; DENIED. Only one allowed, set only by creating a new budget.", nil)
		return
	}
	
	dbBudgetMembership, err := cfg.db.AssignBudgetMemberWithRole(r.Context(), database.AssignBudgetMemberWithRoleParams{
		BudgetID:	pathBudgetID,
		UserID:		parsedUserID,
		MemberRole:	newMemberRole.String(),
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to assign new member to budget", err)
		return
	}

	respBody := BudgetMembership{
		BudgetID: 	dbBudgetMembership.BudgetID,
		UserID: 	dbBudgetMembership.UserID,
		MemberRole:	newMemberRole,
	}
	
	respondWithJSON(w, http.StatusCreated, respBody)
	return

}

func(cfg *apiConfig) endpRemoveBudgetMember(w http.ResponseWriter, r *http.Request){

	pathBudgetID := getContextKeyValue(r.Context(), "budget_id")

	idString := r.PathValue("user_id")
	pathUserID, err := uuid.Parse(idString)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong", err)
		log.Println("Could not parse user_id")
		return
	}
	
	err = cfg.db.RevokeBudgetMembership(r.Context(), database.RevokeBudgetMembershipParams{
		BudgetID:	pathBudgetID,
		UserID:		pathUserID,
	})
	if err != nil {
		respondWithText(w, http.StatusNotFound, "Failed to find membership to revoke")
	}

	respondWithText(w, http.StatusNoContent, "Revoked membership successfully")
	return
}

func(cfg *apiConfig) endpDeleteBudget(w http.ResponseWriter, r *http.Request){
	
	pathBudgetID := getContextKeyValue(r.Context(), "budget_id")

	err := cfg.db.DeleteBudget(r.Context(), pathBudgetID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "404 Not Found", err)
		return
	}

	respondWithText(w, http.StatusNoContent, "Deleted budget")
	return
}