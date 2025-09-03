package main

import (
	"log"
	"strings"
	"net/http"
	"encoding/json"

	"github.com/google/uuid"

	"github.com/YouWantToPinch/pincher-api/internal/database"
	"github.com/YouWantToPinch/pincher-api/internal/auth"
)

func(cfg *apiConfig) endpCreateCategory(w http.ResponseWriter, r *http.Request){
    idString := r.PathValue("user_id")
	pathUserID, err := uuid.Parse(idString)

	type parameters struct {
		Name	string	`json:"name"`
		Notes	string	`json:"notes"`
		GroupID	string	`json:"group_id"`
	}

    decoder := json.NewDecoder(r.Body)
    params := parameters{}
    err = decoder.Decode(&params)
    if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failure decoding parameters", err)
		return
    }

	tokenString, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "No token found to validate", err)
		return
	}
	validatedUserID, err := auth.ValidateJWT(tokenString, cfg.secret)
	if err != nil || validatedUserID != pathUserID {
		respondWithError(w, http.StatusUnauthorized, "401 Unauthorized", err)
		return
	}

	var assignedGroup uuid.NullUUID
	if params.GroupID != "" {
		parsedGroupID, err := uuid.Parse(params.GroupID)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Provided group_id query parameter could not be parsed as UUID", err)
			return
		}
		foundGroup, err := cfg.db.GetGroupByID(r.Context(), parsedGroupID)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Found no group with provided group_id", err)
			return
		}
		assignedGroup.UUID = foundGroup.ID
		assignedGroup.Valid = true
	}

	dbCategory, err := cfg.db.CreateCategory(r.Context(), database.CreateCategoryParams{
		UserID:		validatedUserID,
		GroupID:	assignedGroup,
		Name:		params.Name,
		Notes:		params.Notes,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create category", err)
		return
	}

	respBody := Category{
		ID:			dbCategory.ID,
		CreatedAt:	dbCategory.CreatedAt,
		UpdatedAt:	dbCategory.UpdatedAt,
		UserID:		dbCategory.UserID,
		Name:     	dbCategory.Name,
		GroupID:	dbCategory.GroupID,
		Notes:		dbCategory.Notes,
	}


	respondWithJSON(w, http.StatusCreated, respBody)
	return
}

func(cfg *apiConfig) endpGetCategoriesByUserID(w http.ResponseWriter, r *http.Request) {
	idString := r.PathValue("user_id")
	pathUserID, err := uuid.Parse(idString)
	queryGroupID := r.URL.Query().Get("group_id")
	log.Printf("queryGroupID is: %s", queryGroupID)
	log.Printf("URL: %s", r.URL.String())

	tokenString, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "No token found to validate", err)
		return
	}
	validatedUserID, err := auth.ValidateJWT(tokenString, cfg.secret)
	if err != nil || validatedUserID != pathUserID {
		respondWithError(w, http.StatusUnauthorized, "401 Unauthorized", err)
		return
	}

	categories, err := cfg.db.GetCategoriesByUserID(r.Context(), pathUserID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get user categories", err)
		return
	}

	var respBody []Category
	{
		want := strings.ToLower(strings.TrimSpace(queryGroupID))
		for _, category := range categories {
			if want != "" {
				if !category.GroupID.Valid {
					continue
				}
				got := strings.ToLower(strings.TrimSpace(category.GroupID.UUID.String()))
				if got != want {
					log.Printf("got (%s) != want (%s)", got, want)
					continue
				}
			}
			addCategory := Category{
				ID:			category.ID,
				CreatedAt:	category.CreatedAt,
				UpdatedAt:	category.UpdatedAt,
				Name:     	category.Name,
				UserID:		category.UserID,
				GroupID:	category.GroupID,
				Notes:		category.Notes,
			}
			respBody = append(respBody, addCategory)
		}
	}
	

	respondWithJSON(w, http.StatusOK, respBody)
	return
}

func(cfg *apiConfig) endpAssignCategoryToGroup(w http.ResponseWriter, r *http.Request) {
	idString := r.PathValue("user_id")
	pathUserID, err := uuid.Parse(idString)
	idString = r.PathValue("category_id")
	pathCategoryID, err := uuid.Parse(idString)

	type parameters struct {
		GroupID	string	`json:"group_id"`
	}

    decoder := json.NewDecoder(r.Body)
    params := parameters{}
    err = decoder.Decode(&params)
    if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failure decoding parameters", err)
		return
    }

	tokenString, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "No token found to validate", err)
		return
	}
	validatedUserID, err := auth.ValidateJWT(tokenString, cfg.secret)
	if err != nil || validatedUserID != pathUserID {
		respondWithError(w, http.StatusUnauthorized, "401 Unauthorized", err)
		return
	}

	if params.GroupID == "" {
		respondWithError(w, http.StatusBadRequest, "Request provided no group_id for assignment", err)
		return
	}

	var assignedGroup uuid.NullUUID
	parsedGroupID, err := uuid.Parse(params.GroupID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Provided group_id query parameter could not be parsed as UUID", err)
		return
	}
	foundGroup, err := cfg.db.GetGroupByID(r.Context(), parsedGroupID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Found no group with provided group_id", err)
		return
	}
	assignedGroup.UUID = foundGroup.ID
	assignedGroup.Valid = true

	dbCategory, err := cfg.db.GetCategoryByID(r.Context(), pathCategoryID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Couldn't find category to assign to", err)
		return
	}
	respBody := Category{
		ID:			dbCategory.ID,
		CreatedAt:	dbCategory.CreatedAt,
		UpdatedAt:	dbCategory.UpdatedAt,
		UserID:		dbCategory.UserID,
		Name:     	dbCategory.Name,
		GroupID:	assignedGroup,
		Notes:		dbCategory.Notes,
	}

	cfg.db.AssignCategoryToGroup(r.Context(), database.AssignCategoryToGroupParams{
		ID: 		pathCategoryID,
		GroupID:	assignedGroup,
	})


	respondWithJSON(w, http.StatusCreated, respBody)
	return
}

func(cfg *apiConfig) endpDeleteCategory(w http.ResponseWriter, r *http.Request) {
	idString := r.PathValue("user_id")
	pathUserID, err := uuid.Parse(idString)
	idString = r.PathValue("category_id")
	pathCategoryID, err := uuid.Parse(idString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid id", err)
		return
	}
	
	tokenString, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "No token found to validate", err)
		return
	}
	dbCategory, err := cfg.db.GetCategoryByID(r.Context(), pathCategoryID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Couldn't find category with specified id", err)
		return
	}
	validatedUserID, err := auth.ValidateJWT(tokenString, cfg.secret)
	if err != nil || validatedUserID != pathUserID || validatedUserID != dbCategory.UserID {
		respondWithError(w, http.StatusForbidden, "401 Unauthorized", nil)
		return
	}

	err = cfg.db.DeleteCategoryByID(r.Context(), pathCategoryID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "404 Not Found", err)
		return
	}
	respondWithText(w, http.StatusNoContent, "The category was deleted")
	return
}