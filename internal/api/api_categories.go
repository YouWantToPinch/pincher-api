package api

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/YouWantToPinch/pincher-api/internal/database"
)

func (cfg *APIConfig) handleCreateCategory(w http.ResponseWriter, r *http.Request) {
	type rqSchema struct {
		GroupName string `json:"group_name"`
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

	pathBudgetID := getContextKeyValueAsUUID(r.Context(), "budget_id")

	var assignedGroupID *uuid.UUID
	if rqPayload.GroupName != "" {
		groupID, err := lookupResourceIDByName(r.Context(),
			database.GetBudgetGroupIDByNameParams{
				GroupName: rqPayload.GroupName,
				BudgetID:  pathBudgetID,
			}, cfg.db.GetBudgetGroupIDByName)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "could not get group id", err)
			return
		}
		assignedGroupID = groupID
	}

	dbCategory, err := cfg.db.CreateCategory(r.Context(), database.CreateCategoryParams{
		BudgetID: pathBudgetID,
		GroupID:  assignedGroupID,
		Name:     rqPayload.Name,
		Notes:    rqPayload.Notes,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not create category", err)
		return
	}

	rspPayload := Category{
		ID:        dbCategory.ID,
		CreatedAt: dbCategory.CreatedAt,
		UpdatedAt: dbCategory.UpdatedAt,
		BudgetID:  dbCategory.BudgetID,
		GroupID:   dbCategory.GroupID,
		Meta: Meta{
			Name:  dbCategory.Name,
			Notes: dbCategory.Notes,
		},
	}

	respondWithJSON(w, http.StatusCreated, rspPayload)
}

func (cfg *APIConfig) handleGetCategories(w http.ResponseWriter, r *http.Request) {
	queryGroupID := r.URL.Query().Get("group_id")
	var parsedGroupID uuid.UUID
	if queryGroupID != "" {
		var err error
		parsedGroupID, err = uuid.Parse(queryGroupID)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "could not parse group ID from query", err)
		}
	}

	pathBudgetID := getContextKeyValueAsUUID(r.Context(), "budget_id")

	categories, err := cfg.db.GetCategories(r.Context(), database.GetCategoriesParams{
		BudgetID: pathBudgetID,
		GroupID:  parsedGroupID,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not retrieve categories", err)
		return
	}

	var respCategories []Category
	for _, category := range categories {
		respCategories = append(respCategories, Category{
			ID:        category.ID,
			CreatedAt: category.CreatedAt,
			UpdatedAt: category.UpdatedAt,
			BudgetID:  category.BudgetID,
			GroupID:   category.GroupID,
			Meta: Meta{
				Name:  category.Name,
				Notes: category.Notes,
			},
		})
	}

	type rspSchema struct {
		Categories []Category `json:"categories"`
	}

	rspPayload := rspSchema{
		Categories: respCategories,
	}

	respondWithJSON(w, http.StatusOK, rspPayload)
}

func (cfg *APIConfig) handleUpdateCategory(w http.ResponseWriter, r *http.Request) {
	pathCategoryID, err := parseUUIDFromPath("category_id", r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "", err)
		return
	}

	type rqSchema struct {
		GroupName string `json:"group_name"`
		Meta
	}

	rqPayload, err := decodePayload[rqSchema](r)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "", err)
		return
	}

	pathBudgetID := getContextKeyValueAsUUID(r.Context(), "budget_id")

	var assignedGroupID *uuid.UUID
	if rqPayload.GroupName != "" {
		groupID, err := lookupResourceIDByName(r.Context(),
			database.GetBudgetGroupIDByNameParams{
				GroupName: rqPayload.GroupName,
				BudgetID:  pathBudgetID,
			}, cfg.db.GetBudgetGroupIDByName)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "could not get group id", err)
			return
		}
		assignedGroupID = groupID
	}

	_, err = cfg.db.UpdateCategory(r.Context(), database.UpdateCategoryParams{
		ID:      pathCategoryID,
		GroupID: assignedGroupID,
		Name:    rqPayload.Name,
		Notes:   rqPayload.Notes,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to update category", err)
		return
	}

	respondWithCode(w, http.StatusNoContent)
}

func (cfg *APIConfig) handleDeleteCategory(w http.ResponseWriter, r *http.Request) {
	pathCategoryID, err := parseUUIDFromPath("category_id", r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "", err)
		return
	}

	dbCategory, err := cfg.db.GetCategoryByID(r.Context(), pathCategoryID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "could not get category", err)
		return
	}
	pathBudgetID := getContextKeyValueAsUUID(r.Context(), "budget_id")
	if pathBudgetID != dbCategory.BudgetID {
		respondWithCode(w, http.StatusForbidden)
		return
	}

	err = cfg.db.DeleteCategoryByID(r.Context(), pathCategoryID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "could not delete category", err)
		return
	}
	respondWithCode(w, http.StatusNoContent)
}
