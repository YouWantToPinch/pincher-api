package handler

import (
	"errors"
	"net/http"

	"github.com/YouWantToPinch/pincher-api/internal/service"
)

type UserHandler struct {
	svc service.UserService
}

func NewUserHandler(svc service.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

type UserCredentialPayload struct {
	Password string `json:"password"`
	Username string `json:"username"`
}

func (h *UserHandler) HandleCreateUser(w http.ResponseWriter, r *http.Request) {
	p, err := decodePayload[UserCredentialPayload](r)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "", err)
		return
	}

	user, err := h.svc.RegisterUser(r.Context(), p.Username, p.Password)
	if err == nil {
		respondWithJSON(w, http.StatusCreated, user)
		return
	}
	switch {
	case errors.Is(err, service.ErrUserEmptyCredentials):
		respondWithError(w, http.StatusBadRequest, err.Error(), err)
	default:
		respondWithCode(w, http.StatusInternalServerError)
	}
}

func (h *UserHandler) HandleUpdateUserCredentials(w http.ResponseWriter, r *http.Request) {
	p, err := decodePayload[UserCredentialPayload](r)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "", err)
		return
	}

	err = h.svc.UpdateUserCredentials(r.Context(), p.Username, p.Password)
	if err == nil {
		respondWithCode(w, http.StatusNoContent)
		return
	}
	switch {
	case errors.Is(err, service.ErrUserEmptyCredentials):
		respondWithError(w, http.StatusBadRequest, err.Error(), err)
	case errors.Is(err, service.ErrUserInternalUpdate):
		respondWithError(w, http.StatusConflict, err.Error(), err)
	default:
		respondWithCode(w, http.StatusInternalServerError)
	}
}

func (h *UserHandler) HandleDeleteUser(w http.ResponseWriter, r *http.Request) {
	p, err := decodePayload[UserCredentialPayload](r)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "", err)
		return
	}

	err = h.svc.DeleteUser(r.Context(), p.Username, p.Password)
	if err == nil {
		respondWithCode(w, http.StatusNoContent)
		return
	}
	switch {
	case errors.Is(err, service.ErrUserEmptyCredentials):
		respondWithError(w, http.StatusBadRequest, err.Error(), err)
	case errors.Is(err, service.ErrUserNotFound):
		respondWithError(w, http.StatusBadRequest, err.Error(), err)
	case errors.Is(err, service.ErrUserUnauthorized):
		respondWithError(w, http.StatusUnauthorized, err.Error(), err)
	case errors.Is(err, service.ErrUserForbidden):
		respondWithCode(w, http.StatusForbidden)
	default:
		respondWithCode(w, http.StatusInternalServerError)
	}
}
