package api

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/YouWantToPinch/pincher-api/internal/database"
)

func (cfg *apiConfig) endpCreateCategory(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Name    string `json:"name"`
		Notes   string `json:"notes"`
		GroupID string `json:"group_id"`
	}

	params, err := decodeParams[parameters](r)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failure decoding parameters", err)
		return
	}

	if params.Name == "" {
		respondWithError(w, http.StatusBadRequest, "Name not provided", nil)
		return
	}

	pathBudgetID := getContextKeyValue(r.Context(), "budget_id")

	var assignedGroup uuid.NullUUID
	if params.GroupID != "" {
		parsedGroupID, err := uuid.Parse(params.GroupID)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Provided group_id string could not be parsed as UUID", err)
			return
		}
		foundGroup, err := cfg.db.GetGroupByID(r.Context(), database.GetGroupByIDParams{
			BudgetID: pathBudgetID,
			ID:       parsedGroupID,
		})
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Found no group with provided group_id", err)
			return
		}
		assignedGroup.UUID = foundGroup.ID
		assignedGroup.Valid = true
	}

	dbCategory, err := cfg.db.CreateCategory(r.Context(), database.CreateCategoryParams{
		BudgetID: pathBudgetID,
		GroupID:  assignedGroup,
		Name:     params.Name,
		Notes:    params.Notes,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create category", err)
		return
	}

	respBody := Category{
		ID:        dbCategory.ID,
		CreatedAt: dbCategory.CreatedAt,
		UpdatedAt: dbCategory.UpdatedAt,
		BudgetID:  dbCategory.BudgetID,
		Name:      dbCategory.Name,
		GroupID:   dbCategory.GroupID,
		Notes:     dbCategory.Notes,
	}

	respondWithJSON(w, http.StatusCreated, respBody)
}

func (cfg *apiConfig) endpGetCategories(w http.ResponseWriter, r *http.Request) {
	queryGroupID := r.URL.Query().Get("group_id")
	var parsedGroupID uuid.UUID
	if queryGroupID != "" {
		var err error
		parsedGroupID, err = uuid.Parse(queryGroupID)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "invalid id", err)
		}
	}

	pathBudgetID := getContextKeyValue(r.Context(), "budget_id")

	categories, err := cfg.db.GetCategories(r.Context(), database.GetCategoriesParams{
		BudgetID: pathBudgetID,
		GroupID:  parsedGroupID,
	})
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Database error", err)
		return
	}

	var respCategories []Category
	for _, category := range categories {
		respCategories = append(respCategories, Category{
			ID:        category.ID,
			CreatedAt: category.CreatedAt,
			UpdatedAt: category.UpdatedAt,
			Name:      category.Name,
			BudgetID:  category.BudgetID,
			GroupID:   category.GroupID,
			Notes:     category.Notes,
		})
	}

	type resp struct {
		Categories []Category `json:"categories"`
	}

	respBody := resp{
		Categories: respCategories,
	}

	respondWithJSON(w, http.StatusOK, respBody)
}

func (cfg *apiConfig) endpUpdateCategory(w http.ResponseWriter, r *http.Request) {
	var pathCategoryID uuid.UUID
	err := parseUUIDFromPath("category_id", r, &pathCategoryID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid id", err)
		return
	}

	type parameters struct {
		Name    string `json:"name"`
		Notes   string `json:"notes"`
		GroupID string `json:"group_id"`
	}

	params, err := decodeParams[parameters](r)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failure decoding parameters", err)
		return
	}

	pathBudgetID := getContextKeyValue(r.Context(), "budget_id")

	var assignedGroup uuid.NullUUID
	if params.GroupID != "" {
		parsedGroupID, err := uuid.Parse(params.GroupID)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Provided group_id could not be parsed as UUID", err)
			return
		}
		foundGroup, err := cfg.db.GetGroupByID(r.Context(), database.GetGroupByIDParams{
			BudgetID: pathBudgetID,
			ID:       parsedGroupID,
		})
		if err != nil || pathBudgetID != foundGroup.BudgetID {
			respondWithError(w, http.StatusBadRequest, "Found no group in budget with provided group_id", err)
			return
		}
		assignedGroup.UUID = foundGroup.ID
		assignedGroup.Valid = true
	} else {
		assignedGroup.Valid = false
	}

	_, err = cfg.db.UpdateCategory(r.Context(), database.UpdateCategoryParams{
		ID:      pathCategoryID,
		GroupID: assignedGroup,
		Name:    params.Name,
		Notes:   params.Notes,
	})
	if err != nil {
		respondWithError(w, http.StatusNotModified, "Failed to update category", err)
		return
	}

	respondWithText(w, http.StatusNoContent, "Category '"+params.Name+"' updated successfully!")
}

func (cfg *apiConfig) endpDeleteCategory(w http.ResponseWriter, r *http.Request) {
	var pathCategoryID uuid.UUID
	err := parseUUIDFromPath("category_id", r, &pathCategoryID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid id", err)
		return
	}

	dbCategory, err := cfg.db.GetCategoryByID(r.Context(), pathCategoryID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Couldn't find category with specified id", err)
		return
	}
	pathBudgetID := getContextKeyValue(r.Context(), "budget_id")
	if pathBudgetID != dbCategory.BudgetID {
		respondWithError(w, http.StatusForbidden, "401 Unauthorized", nil)
		return
	}

	err = cfg.db.DeleteCategoryByID(r.Context(), pathCategoryID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "404 Not Found", err)
		return
	}
	respondWithText(w, http.StatusNoContent, "The category was deleted")
}
