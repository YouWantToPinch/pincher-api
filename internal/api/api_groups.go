package api

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/YouWantToPinch/pincher-api/internal/database"
)

func (cfg *APIConfig) endpCreateGroup(w http.ResponseWriter, r *http.Request) {
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

	pathBudgetID := getContextKeyValue(r.Context(), "budget_id")

	_, err = cfg.db.GetGroupByBudgetIDAndName(r.Context(), database.GetGroupByBudgetIDAndNameParams{
		Name:     rqPayload.Name,
		BudgetID: pathBudgetID,
	})
	if err == nil {
		respondWithError(w, http.StatusConflict, "a group with provided name already exists", err)
		return
	}

	dbGroup, err := cfg.db.CreateGroup(r.Context(), database.CreateGroupParams{
		BudgetID: pathBudgetID,
		Name:     rqPayload.Name,
		Notes:    rqPayload.Notes,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not create group", err)
		return
	}
	rspPayload := Group{
		ID:        dbGroup.ID,
		CreatedAt: dbGroup.CreatedAt,
		UpdatedAt: dbGroup.UpdatedAt,
		BudgetID:  dbGroup.BudgetID,
		Meta: Meta{
			Name:  dbGroup.Name,
			Notes: dbGroup.Notes,
		},
	}

	respondWithJSON(w, http.StatusCreated, rspPayload)
}

func (cfg *APIConfig) endpGetGroups(w http.ResponseWriter, r *http.Request) {
	pathBudgetID := getContextKeyValue(r.Context(), "budget_id")

	dbGroups, err := cfg.db.GetGroupsByBudgetID(r.Context(), pathBudgetID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not get user category groups", err)
		return
	}

	var groups []Group
	for _, dbGroup := range dbGroups {
		groups = append(groups, Group{
			ID:        dbGroup.ID,
			CreatedAt: dbGroup.CreatedAt,
			UpdatedAt: dbGroup.UpdatedAt,
			BudgetID:  dbGroup.BudgetID,
			Meta: Meta{
				Name:  dbGroup.Name,
				Notes: dbGroup.Notes,
			},
		})
	}

	type rspSchema struct {
		Groups []Group `json:"groups"`
	}

	rspPayload := rspSchema{
		Groups: groups,
	}

	respondWithJSON(w, http.StatusOK, rspPayload)
}

func (cfg *APIConfig) endpUpdateGroup(w http.ResponseWriter, r *http.Request) {
	var pathGroupID uuid.UUID
	err := parseUUIDFromPath("group_id", r, &pathGroupID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "", err)
		return
	}

	type rqSchema struct {
		Meta
	}

	rqPayload, err := decodePayload[rqSchema](r)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "", err)
		return
	}

	_, err = cfg.db.UpdateGroup(r.Context(), database.UpdateGroupParams{
		ID:    pathGroupID,
		Name:  rqPayload.Name,
		Notes: rqPayload.Notes,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not update group", err)
		return
	}

	respondWithCode(w, http.StatusNoContent)
}

func (cfg *APIConfig) endpDeleteGroup(w http.ResponseWriter, r *http.Request) {
	var pathGroupID uuid.UUID
	err := parseUUIDFromPath("group_id", r, &pathGroupID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "", err)
		return
	}

	pathBudgetID := getContextKeyValue(r.Context(), "budget_id")

	dbGroup, err := cfg.db.GetGroupByID(r.Context(), database.GetGroupByIDParams{
		BudgetID: pathBudgetID,
		ID:       pathGroupID,
	})
	if err != nil {
		respondWithError(w, http.StatusNotFound, "could not find group", err)
		return
	}
	if pathBudgetID != dbGroup.BudgetID {
		respondWithCode(w, http.StatusForbidden)
		return
	}

	err = cfg.db.DeleteGroupByID(r.Context(), pathGroupID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not delete group", err)
		return
	}

	respondWithCode(w, http.StatusNoContent)
}
