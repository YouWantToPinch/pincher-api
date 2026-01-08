package api

import (
	"net/http"

	"github.com/YouWantToPinch/pincher-api/internal/auth"
	"github.com/YouWantToPinch/pincher-api/internal/database"
)

func (cfg *APIConfig) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	type rqSchema struct {
		Password string `json:"password"`
		Username string `json:"username"`
	}

	rqPayload, err := decodePayload[rqSchema](r)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "", err)
		return
	}

	if rqPayload.Username == "" || rqPayload.Password == "" {
		respondWithError(w, http.StatusBadRequest, "missing username or password", nil)
		return
	}

	hashedPass, err := auth.HashPassword(rqPayload.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failure processing request to create user", err)
		return
	}

	dbUser, err := cfg.db.CreateUser(r.Context(), database.CreateUserParams{
		Username:       rqPayload.Username,
		HashedPassword: hashedPass,
	})
	if err != nil {
		respondWithError(w, http.StatusConflict, "failure processing request to create user", err)
		return
	}
	rspPayload := User{
		ID:        dbUser.ID,
		CreatedAt: dbUser.CreatedAt,
		UpdatedAt: dbUser.UpdatedAt,
		Username:  dbUser.Username,
	}

	respondWithJSON(w, http.StatusCreated, rspPayload)
}

func (cfg *APIConfig) handleUpdateUserCredentials(w http.ResponseWriter, r *http.Request) {
	type rqSchema struct {
		Password string `json:"password"`
		Username string `json:"username"`
	}

	rqPayload, err := decodePayload[rqSchema](r)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "", err)
		return
	}

	if rqPayload.Username == "" || rqPayload.Password == "" {
		respondWithError(w, http.StatusBadRequest, "missing username or password", nil)
		return
	}

	hashedPass, err := auth.HashPassword(rqPayload.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failure processing request to create user", err)
		return
	}

	validatedUserID := getContextKeyValueAsUUID(r.Context(), "user_id")

	_, err = cfg.db.UpdateUserCredentials(r.Context(), database.UpdateUserCredentialsParams{
		ID:             validatedUserID,
		Username:       rqPayload.Username,
		HashedPassword: hashedPass,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not update user credentials", err)
	}

	respondWithCode(w, http.StatusNoContent)
}

func (cfg *APIConfig) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	type rqSchema struct {
		Password string `json:"password"`
		Username string `json:"username"`
	}

	rqPayload, err := decodePayload[rqSchema](r)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "", err)
		return
	}

	if rqPayload.Username == "" || rqPayload.Password == "" {
		respondWithError(w, http.StatusBadRequest, "missing username or password", nil)
		return
	}

	validatedUserID := getContextKeyValueAsUUID(r.Context(), "user_id")

	dbUser, err := cfg.db.GetUserByUsername(r.Context(), rqPayload.Username)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "incorrect username or password", err)
		return
	}
	if validatedUserID != dbUser.ID {
		respondWithCode(w, http.StatusForbidden)
		return
	}

	match, err := auth.CheckPasswordHash(rqPayload.Password, dbUser.HashedPassword)
	if err != nil || !match {
		respondWithError(w, http.StatusUnauthorized, "incorrect username or password", err)
		return
	}

	err = cfg.db.DeleteUserByID(r.Context(), validatedUserID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not delete user", err)
		return
	}

	respondWithCode(w, http.StatusNoContent)
}
