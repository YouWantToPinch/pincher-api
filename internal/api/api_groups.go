package api

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/YouWantToPinch/pincher-api/internal/database"
)

func (cfg *apiConfig) endpCreateGroup(w http.ResponseWriter, r *http.Request) {

	type parameters struct {
		Name  string `json:"name"`
		Notes string `json:"notes"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failure decoding parameters", err)
		return
	}

	pathBudgetID := getContextKeyValue(r.Context(), "budget_id")

	_, err = cfg.db.GetGroupByBudgetIDAndName(r.Context(), database.GetGroupByBudgetIDAndNameParams{
		Name:     params.Name,
		BudgetID: pathBudgetID,
	})
	if err == nil {
		respondWithError(w, http.StatusConflict, "Group already exists for user", err)
		return
	}

	dbGroup, err := cfg.db.CreateGroup(r.Context(), database.CreateGroupParams{
		BudgetID: pathBudgetID,
		Name:     params.Name,
		Notes:    params.Notes,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create group", err)
		return
	}
	respBody := Group{
		ID:        dbGroup.ID,
		CreatedAt: dbGroup.CreatedAt,
		UpdatedAt: dbGroup.UpdatedAt,
		BudgetID:  dbGroup.BudgetID,
		Name:      dbGroup.Name,
		Notes:     dbGroup.Notes,
	}

	respondWithJSON(w, http.StatusCreated, respBody)
}

func (cfg *apiConfig) endpGetGroups(w http.ResponseWriter, r *http.Request) {

	pathBudgetID := getContextKeyValue(r.Context(), "budget_id")

	dbGroups, err := cfg.db.GetGroupsByBudgetID(r.Context(), pathBudgetID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get user category groups", err)
		return
	}

	var respBody []Group
	for _, group := range dbGroups {
		addGroup := Group{
			ID:        group.ID,
			CreatedAt: group.CreatedAt,
			UpdatedAt: group.UpdatedAt,
			Name:      group.Name,
			BudgetID:  group.BudgetID,
			Notes:     group.Notes,
		}
		respBody = append(respBody, addGroup)
	}

	respondWithJSON(w, http.StatusOK, respBody)
}

func (cfg *apiConfig) endpDeleteGroup(w http.ResponseWriter, r *http.Request) {
	idString := r.PathValue("group_id")
	pathGroupID, err := uuid.Parse(idString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid id", err)
		return
	}

	pathBudgetID := getContextKeyValue(r.Context(), "budget_id")

	dbGroup, err := cfg.db.GetGroupByID(r.Context(), database.GetGroupByIDParams{
		BudgetID: pathBudgetID,
		ID:       pathGroupID,
	})
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Couldn't find group with specified id", err)
		return
	}
	if pathBudgetID != dbGroup.BudgetID {
		respondWithError(w, http.StatusForbidden, "401 Unauthorized", nil)
		return
	}

	err = cfg.db.DeleteGroupByID(r.Context(), pathGroupID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "404 Not Found", err)
		return
	}
	respondWithText(w, http.StatusNoContent, "The group was deleted")
}
