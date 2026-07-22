package handler

import (
	"errors"
	"net/http"

	"github.com/YouWantToPinch/pincher-api/internal/models"
	"github.com/YouWantToPinch/pincher-api/internal/service"
)

type CategoryHandler struct {
	svc service.CategoryService
}

func NewCategoryHandler(svc service.CategoryService) *CategoryHandler {
	return &CategoryHandler{svc: svc}
}

type CategoryUpsertPayload struct {
	GroupName string `json:"group_name"`
	Name      string `json:"name"`
	Notes     string `json:"notes"`
}

func (h *CategoryHandler) HandleCreateCategory(w http.ResponseWriter, r *http.Request) {
	p, err := decodePayload[CategoryUpsertPayload](r)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "", err)
		return
	}

	category, err := h.svc.CreateCategory(r.Context(), p.Name, p.Notes, p.GroupName)
	if err == nil {
		respondWithJSON(w, http.StatusCreated, category)
		return
	}
	switch {
	case errors.Is(err, service.ErrCategoryInvalidInputName):
		respondWithError(w, http.StatusBadRequest, err.Error(), err)
	case errors.Is(err, service.ErrCategoryInternalCreate):
		respondWithError(w, http.StatusConflict, err.Error(), err)
	default:
		respondWithCode(w, http.StatusInternalServerError)
	}
}

func (h *CategoryHandler) HandleGetCategories(w http.ResponseWriter, r *http.Request) {
	type rspSchema struct {
		Categories []models.Category `json:"data"`
	}

	groupNameQuery := r.URL.Query().Get("group_name")

	categories, err := h.svc.GetCategories(r.Context(), groupNameQuery)
	if err == nil {
		respondWithJSON(w, http.StatusOK, rspSchema{
			Categories: categories,
		})
	}

	respondWithCode(w, http.StatusInternalServerError)
}

func (h *CategoryHandler) HandleGetCategory(w http.ResponseWriter, r *http.Request) {
	type rspSchema struct {
		Categories []models.Category `json:"data"`
	}

	categoryID, err := parseUUIDFromPath("category_id", r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "", err)
		return
	}

	category, err := h.svc.GetCategoryByID(r.Context(), categoryID)
	if err == nil {
		respondWithJSON(w, http.StatusOK, category)
		return
	}

	respondWithCode(w, http.StatusInternalServerError)
}

func (h *CategoryHandler) HandleUpdateCategory(w http.ResponseWriter, r *http.Request) {
	p, err := decodePayload[CategoryUpsertPayload](r)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "", err)
		return
	}

	categoryID, err := parseUUIDFromPath("category_id", r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "", err)
		return
	}

	err = h.svc.UpdateCategory(r.Context(), p.Name, p.Notes, p.GroupName, categoryID)
	if err == nil {
		respondWithCode(w, http.StatusNoContent)
		return
	}
	switch {
	case errors.Is(err, service.ErrCategoryInvalidInputName):
		respondWithError(w, http.StatusBadRequest, err.Error(), err)
	case errors.Is(err, service.ErrCategoryInternalUpdate):
		respondWithError(w, http.StatusConflict, err.Error(), err)
	default:
		respondWithCode(w, http.StatusInternalServerError)
	}
}

type CategoryDeletePayload struct {
	NewCategoryName string `json:"new_category_name"`
}

func (h *CategoryHandler) HandleDeleteCategory(w http.ResponseWriter, r *http.Request) {
	categoryID, err := parseUUIDFromPath("category_id", r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "", err)
		return
	}

	rqPayload, err := decodePayload[CategoryDeletePayload](r)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "", err)
		return
	}

	err = h.svc.DeleteCategory(r.Context(), categoryID, rqPayload.NewCategoryName)
	if err == nil {
		respondWithCode(w, http.StatusNoContent)
		return
	}
	switch {
	case errors.Is(err, service.ErrCategoryInvalidInputName), errors.Is(err, service.ErrCategoryInvalidInputID):
		respondWithError(w, http.StatusBadRequest, err.Error(), err)
	case errors.Is(err, service.ErrCategoryNotFound):
		respondWithError(w, http.StatusBadRequest, err.Error(), err)
	case errors.Is(err, service.ErrCategoryForbidden):
		respondWithCode(w, http.StatusForbidden)
	default:
		respondWithCode(w, http.StatusInternalServerError)
	}
}
