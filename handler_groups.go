package main

import (
	"net/http"
	"encoding/json"

	"github.com/google/uuid"

	"github.com/YouWantToPinch/pincher-api/internal/database"
)

func(cfg *apiConfig) endpCreateGroup(w http.ResponseWriter, r *http.Request){

	type parameters struct {
		Name	string `json:"name"`
		Notes	string `json:"notes"`
	}

    decoder := json.NewDecoder(r.Body)
    params := parameters{}
    err := decoder.Decode(&params)
    if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failure decoding parameters", err)
		return
    }

	validatedUserID := getValidatedUserID(r.Context())

	_, err = cfg.db.GetGroupByUserIDAndName(r.Context(), database.GetGroupByUserIDAndNameParams{
		Name: params.Name,
		UserID: validatedUserID,
	})
	if err == nil {
		respondWithError(w, http.StatusConflict, "Group already exists for user", err)
		return
	}

	dbGroup, err := cfg.db.CreateGroup(r.Context(), database.CreateGroupParams{
		UserID:	validatedUserID,
		Name:	params.Name,
		Notes:	params.Notes,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create group", err)
		return
	}
	respBody := Group{
		ID:			dbGroup.ID,
		CreatedAt:	dbGroup.CreatedAt,
		UpdatedAt:	dbGroup.UpdatedAt,
		UserID:		dbGroup.UserID,
		Name:     	dbGroup.Name,
		Notes:		dbGroup.Notes,
	}

	respondWithJSON(w, http.StatusCreated, respBody)
	return
}

func(cfg *apiConfig) endpGetGroups(w http.ResponseWriter, r *http.Request) {

	validatedUserID := getValidatedUserID(r.Context())

	groups, err := cfg.db.GetGroupsByUserID(r.Context(), validatedUserID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get user category groups", err)
		return
	}

	var respBody []Group
	for _, group := range groups {
		addGroup := Group{
			ID:			group.ID,
			CreatedAt:	group.CreatedAt,
			UpdatedAt:	group.UpdatedAt,
			Name:     	group.Name,
			UserID:		group.UserID,
			Notes:		group.Notes,
		}
		respBody = append(respBody, addGroup)
	}
	
	respondWithJSON(w, http.StatusOK, respBody)
	return
}

func(cfg *apiConfig) endpDeleteGroup(w http.ResponseWriter, r *http.Request) {
	idString := r.PathValue("group_id")
	pathGroupID, err := uuid.Parse(idString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid id", err)
		return
	}
	
	validatedUserID := getValidatedUserID(r.Context())

	dbGroup, err := cfg.db.GetGroupByID(r.Context(), pathGroupID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Couldn't find group with specified id", err)
		return
	}
	if validatedUserID != dbGroup.UserID {
		respondWithError(w, http.StatusForbidden, "401 Unauthorized", nil)
		return
	}

	err = cfg.db.DeleteGroupByID(r.Context(), pathGroupID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "404 Not Found", err)
		return
	}
	respondWithText(w, http.StatusNoContent, "The group was deleted")
	return
}