package api

import (
	"net/http"

	"github.com/google/uuid"

	db "github.com/YouWantToPinch/pincher-api/internal/database"
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
			db.GetBudgetGroupIDByNameParams{
				GroupName: rqPayload.GroupName,
				BudgetID:  pathBudgetID,
			}, cfg.db.GetBudgetGroupIDByName)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "could not get group id", err)
			return
		}
		if groupID != uuid.Nil {
			assignedGroupID = &groupID
		}
	}

	dbCategory, err := cfg.db.CreateCategory(r.Context(), db.CreateCategoryParams{
		BudgetID: pathBudgetID,
		GroupID:  assignedGroupID,
		Name:     rqPayload.Name,
		Notes:    rqPayload.Notes,
	})
	if err != nil {
		respondWithError(w, http.StatusConflict, "could not create category", err)
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
	pathBudgetID := getContextKeyValueAsUUID(r.Context(), "budget_id")

	var err error

	groupNameQuery := r.URL.Query().Get("group_name")
	parsedGroupID := uuid.Nil
	if groupNameQuery != "" {
		parsedGroupID, err = lookupResourceIDByName(r.Context(),
			db.GetBudgetGroupIDByNameParams{
				GroupName: groupNameQuery,
				BudgetID:  pathBudgetID,
			}, cfg.db.GetBudgetGroupIDByName)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "could not get account id", err)
			return
		}
	}

	categories, err := cfg.db.GetCategories(r.Context(), db.GetCategoriesParams{
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
		Categories []Category `json:"data"`
	}

	rspPayload := rspSchema{
		Categories: respCategories,
	}

	respondWithJSON(w, http.StatusOK, rspPayload)
}

func (cfg *APIConfig) handleGetCategory(w http.ResponseWriter, r *http.Request) {
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

	rspPayload := Category{
		ID:        dbCategory.ID,
		CreatedAt: dbCategory.CreatedAt,
		UpdatedAt: dbCategory.UpdatedAt,
		BudgetID:  dbCategory.BudgetID,
		Meta: Meta{
			Name:  dbCategory.Name,
			Notes: dbCategory.Notes,
		},
	}

	respondWithJSON(w, http.StatusCreated, rspPayload)
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
			db.GetBudgetGroupIDByNameParams{
				GroupName: rqPayload.GroupName,
				BudgetID:  pathBudgetID,
			}, cfg.db.GetBudgetGroupIDByName)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "could not get group id", err)
			return
		}
		if groupID != uuid.Nil {
			assignedGroupID = &groupID
		}
	}

	_, err = cfg.db.UpdateCategory(r.Context(), db.UpdateCategoryParams{
		ID:      pathCategoryID,
		GroupID: assignedGroupID,
		Name:    rqPayload.Name,
		Notes:   rqPayload.Notes,
	})
	if err != nil {
		respondWithError(w, http.StatusConflict, "failed to update category", err)
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

	// DB TRANSACTION BLOCK
	{
		tx, err := cfg.Pool.Begin(r.Context())
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "", err)
			return
		}
		defer tx.Rollback(r.Context())

		q := cfg.db.WithTx(tx)

		dbCategory, err := q.GetCategoryByID(r.Context(), pathCategoryID)
		if err != nil {
			respondWithError(w, http.StatusNotFound, "could not get category", err)
			return
		}

		pathBudgetID := getContextKeyValueAsUUID(r.Context(), "budget_id")
		if pathBudgetID != dbCategory.BudgetID {
			respondWithCode(w, http.StatusForbidden)
			return
		}

		categoryInUse, err := q.IsCategoryInUse(r.Context(), &pathCategoryID)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "could not determine whether category in use", err)
		}

		if categoryInUse {

			type rqSchema struct {
				NewCategoryName string `json:"new_category_name"`
			}

			rqPayload, err := decodePayload[rqSchema](r)
			if err != nil {
				respondWithError(w, http.StatusInternalServerError, "", err)
				return
			}

			if rqPayload.NewCategoryName == "" {
				respondWithError(w, http.StatusUnprocessableEntity, "replacement category name not provided", nil)
				return
			}
			categoryID, err := lookupResourceIDByName(r.Context(),
				db.GetBudgetCategoryIDByNameParams{
					CategoryName: rqPayload.NewCategoryName,
					BudgetID:     pathBudgetID,
				}, q.GetBudgetCategoryIDByName)
			if err != nil {
				respondWithError(w, http.StatusBadRequest, "no category found for replacement with given name", err)
				return
			}

			err = q.ReassignTransactionCategories(r.Context(), db.ReassignTransactionCategoriesParams{
				OldCategoryID: pathCategoryID,
				NewCategoryID: categoryID,
			})
			if err != nil {
				respondWithError(w, http.StatusInternalServerError, "could not reassign category for transactions", err)
				return
			}

			err = q.ReassignAssignmentCategories(r.Context(), db.ReassignAssignmentCategoriesParams{
				OldCategoryID: pathCategoryID,
				NewCategoryID: categoryID,
			})
			if err != nil {
				respondWithError(w, http.StatusInternalServerError, "could not reassign category for assignments", err)
				return
			}
		}

		err = q.DeleteCategory(r.Context(), pathCategoryID)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "could not delete category", err)
			return
		}
		if err := tx.Commit(r.Context()); err != nil {
			respondWithError(w, http.StatusInternalServerError, "", err)
			return
		}
	}

	respondWithCode(w, http.StatusNoContent)
}
