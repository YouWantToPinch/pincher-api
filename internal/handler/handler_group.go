package handler

import (
	"errors"
	"net/http"

	"github.com/YouWantToPinch/pincher-api/internal/models"
	"github.com/YouWantToPinch/pincher-api/internal/service"
)

type GroupHandler struct {
	svc service.GroupService
}

func NewGroupHandler(svc service.GroupService) *GroupHandler {
	return &GroupHandler{svc: svc}
}

type GroupUpsertPayload struct {
	Name  string `json:"name"`
	Notes string `json:"notes"`
}

func (h *GroupHandler) HandleCreateGroup(w http.ResponseWriter, r *http.Request) {
	p, err := decodePayload[GroupUpsertPayload](r)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "", err)
		return
	}

	group, err := h.svc.CreateGroup(r.Context(), p.Name, p.Notes)
	if err == nil {
		respondWithJSON(w, http.StatusCreated, group)
		return
	}
	switch {
	case errors.Is(err, service.ErrGroupInvalidInputName):
		respondWithError(w, http.StatusBadRequest, err.Error(), err)
	case errors.Is(err, service.ErrGroupInternalCreate):
		respondWithError(w, http.StatusConflict, err.Error(), err)
	default:
		respondWithCode(w, http.StatusInternalServerError)
	}
}

func (h *GroupHandler) HandleGetGroups(w http.ResponseWriter, r *http.Request) {
	type rspSchema struct {
		Groups []models.Group `json:"data"`
	}

	groups, err := h.svc.GetGroups(r.Context())
	if err == nil {
		respondWithJSON(w, http.StatusOK, rspSchema{
			Groups: groups,
		})
	}

	respondWithCode(w, http.StatusInternalServerError)
}

func (h *GroupHandler) HandleGetGroup(w http.ResponseWriter, r *http.Request) {
	type rspSchema struct {
		Groups []models.Group `json:"data"`
	}

	groupID, err := parseUUIDFromPath("group_id", r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "", err)
		return
	}

	group, err := h.svc.GetGroupByID(r.Context(), groupID)
	if err == nil {
		respondWithJSON(w, http.StatusOK, group)
		return
	}

	respondWithCode(w, http.StatusInternalServerError)
}

func (h *GroupHandler) HandleUpdateGroup(w http.ResponseWriter, r *http.Request) {
	p, err := decodePayload[GroupUpsertPayload](r)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "", err)
		return
	}

	groupID, err := parseUUIDFromPath("group_id", r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "", err)
		return
	}

	err = h.svc.UpdateGroup(r.Context(), p.Name, p.Notes, groupID)
	if err == nil {
		respondWithCode(w, http.StatusNoContent)
		return
	}
	switch {
	case errors.Is(err, service.ErrGroupInvalidInputName):
		respondWithError(w, http.StatusBadRequest, err.Error(), err)
	case errors.Is(err, service.ErrGroupInternalUpdate):
		respondWithError(w, http.StatusConflict, err.Error(), err)
	default:
		respondWithCode(w, http.StatusInternalServerError)
	}
}

func (h *GroupHandler) HandleDeleteGroup(w http.ResponseWriter, r *http.Request) {
	groupID, err := parseUUIDFromPath("group_id", r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "", err)
		return
	}

	err = h.svc.DeleteGroupByID(r.Context(), groupID)
	if err == nil {
		respondWithCode(w, http.StatusNoContent)
		return
	}
	switch {
	case errors.Is(err, service.ErrGroupInvalidInputName), errors.Is(err, service.ErrGroupInvalidInputID):
		respondWithError(w, http.StatusBadRequest, err.Error(), err)
	case errors.Is(err, service.ErrGroupNotFound):
		respondWithError(w, http.StatusBadRequest, err.Error(), err)
	case errors.Is(err, service.ErrGroupForbidden):
		respondWithCode(w, http.StatusForbidden)
	default:
		respondWithCode(w, http.StatusInternalServerError)
	}
}
