package api

import (
	"fmt"
	"github.com/google/uuid"
	"log/slog"
	"net/http"
	"strings"

	"github.com/YouWantToPinch/pincher-api/internal/database"
)

func (cfg *apiConfig) endpCreateCategory(w http.ResponseWriter, r *http.Request) {

	type parameters struct {
		Name    string `json:"name"`
		Notes   string `json:"notes"`
		GroupID string `json:"group_id"`
	}

	var params parameters
	err := decodeParams(r, &params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failure decoding parameters", err)
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
	slog.Info(fmt.Sprintf("queryGroupID is: %s", queryGroupID))
	slog.Info(fmt.Sprintf("URL: %s", r.URL.String()))

	pathBudgetID := getContextKeyValue(r.Context(), "budget_id")

	categories, err := cfg.db.GetCategoriesByBudgetID(r.Context(), pathBudgetID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Couldn't find categories in group specified", err)
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
					slog.Info(fmt.Sprintf("got (%s) != want (%s)", got, want))
					continue
				}
			}
			addCategory := Category{
				ID:        category.ID,
				CreatedAt: category.CreatedAt,
				UpdatedAt: category.UpdatedAt,
				Name:      category.Name,
				BudgetID:  category.BudgetID,
				GroupID:   category.GroupID,
				Notes:     category.Notes,
			}
			respBody = append(respBody, addCategory)
		}
	}

	respondWithJSON(w, http.StatusOK, respBody)
}

func (cfg *apiConfig) endpAssignCategoryToGroup(w http.ResponseWriter, r *http.Request) {
	idString := r.PathValue("category_id")
	pathCategoryID, err := uuid.Parse(idString)

	type parameters struct {
		GroupID string `json:"group_id"`
	}

	var params parameters
	err = decodeParams(r, &params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failure decoding parameters", err)
		return
	}

	pathBudgetID := getContextKeyValue(r.Context(), "budget_id")

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
	foundGroup, err := cfg.db.GetGroupByID(r.Context(), database.GetGroupByIDParams{
		BudgetID: pathBudgetID,
		ID:       parsedGroupID,
	})
	if err != nil || pathBudgetID != foundGroup.BudgetID {
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
		ID:        dbCategory.ID,
		CreatedAt: dbCategory.CreatedAt,
		UpdatedAt: dbCategory.UpdatedAt,
		BudgetID:  dbCategory.BudgetID,
		Name:      dbCategory.Name,
		GroupID:   assignedGroup,
		Notes:     dbCategory.Notes,
	}

	cfg.db.AssignCategoryToGroup(r.Context(), database.AssignCategoryToGroupParams{
		ID:      pathCategoryID,
		GroupID: assignedGroup,
	})

	respondWithJSON(w, http.StatusCreated, respBody)
}

func (cfg *apiConfig) endpDeleteCategory(w http.ResponseWriter, r *http.Request) {
	idString := r.PathValue("category_id")
	pathCategoryID, err := uuid.Parse(idString)
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
