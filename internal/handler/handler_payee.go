package handler

import (
	"errors"
	"net/http"

	"github.com/YouWantToPinch/pincher-api/internal/models"
	"github.com/YouWantToPinch/pincher-api/internal/service"
)

type PayeeHandler struct {
	svc service.PayeeService
}

func NewPayeeHandler(svc service.PayeeService) *PayeeHandler {
	return &PayeeHandler{svc: svc}
}

type PayeeUpsertPayload struct {
	Name  string `json:"name"`
	Notes string `json:"notes"`
}

func (h *PayeeHandler) HandleCreatePayee(w http.ResponseWriter, r *http.Request) {
	p, err := decodePayload[PayeeUpsertPayload](r)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "", err)
		return
	}

	payee, err := h.svc.CreatePayee(r.Context(), p.Name, p.Notes)
	if err == nil {
		respondWithJSON(w, http.StatusCreated, payee)
		return
	}
	switch {
	case errors.Is(err, service.ErrPayeeInvalidInputName):
		respondWithError(w, http.StatusBadRequest, err.Error(), err)
	case errors.Is(err, service.ErrPayeeInternalCreate):
		respondWithError(w, http.StatusConflict, err.Error(), err)
	default:
		respondWithCode(w, http.StatusInternalServerError)
	}
}

func (h *PayeeHandler) HandleGetPayees(w http.ResponseWriter, r *http.Request) {
	type rspSchema struct {
		Payees []models.Payee `json:"data"`
	}

	payees, err := h.svc.GetPayees(r.Context())
	if err == nil {
		respondWithJSON(w, http.StatusOK, rspSchema{
			Payees: payees,
		})
	}

	respondWithCode(w, http.StatusInternalServerError)
}

func (h *PayeeHandler) HandleGetPayee(w http.ResponseWriter, r *http.Request) {
	type rspSchema struct {
		Payees []models.Payee `json:"data"`
	}

	payeeID, err := parseUUIDFromPath("payee_id", r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "", err)
		return
	}

	payee, err := h.svc.GetPayeeByID(r.Context(), payeeID)
	if err == nil {
		respondWithJSON(w, http.StatusOK, payee)
		return
	}

	respondWithCode(w, http.StatusInternalServerError)
}

func (h *PayeeHandler) HandleUpdatePayee(w http.ResponseWriter, r *http.Request) {
	p, err := decodePayload[PayeeUpsertPayload](r)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "", err)
		return
	}

	payeeID, err := parseUUIDFromPath("payee_id", r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "", err)
		return
	}

	err = h.svc.UpdatePayee(r.Context(), p.Name, p.Notes, payeeID)
	if err == nil {
		respondWithCode(w, http.StatusNoContent)
		return
	}
	switch {
	case errors.Is(err, service.ErrPayeeInvalidInputName):
		respondWithError(w, http.StatusBadRequest, err.Error(), err)
	case errors.Is(err, service.ErrPayeeInternalUpdate):
		respondWithError(w, http.StatusConflict, err.Error(), err)
	default:
		respondWithCode(w, http.StatusInternalServerError)
	}
}

type PayeeDeletePayload struct {
	NewPayeeName string `json:"new_payee_name"`
}

func (h *PayeeHandler) HandleDeletePayee(w http.ResponseWriter, r *http.Request) {
	payeeID, err := parseUUIDFromPath("payee_id", r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "", err)
		return
	}

	rqPayload, err := decodePayload[PayeeDeletePayload](r)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "", err)
		return
	}

	err = h.svc.DeletePayeeByID(r.Context(), payeeID, rqPayload.NewPayeeName)
	if err == nil {
		respondWithCode(w, http.StatusNoContent)
		return
	}
	switch {
	case errors.Is(err, service.ErrPayeeInvalidInputName), errors.Is(err, service.ErrPayeeInvalidInputID):
		respondWithError(w, http.StatusBadRequest, err.Error(), err)
	case errors.Is(err, service.ErrPayeeNotFound):
		respondWithError(w, http.StatusBadRequest, err.Error(), err)
	case errors.Is(err, service.ErrPayeeForbidden):
		respondWithCode(w, http.StatusForbidden)
	default:
		respondWithCode(w, http.StatusInternalServerError)
	}
}
