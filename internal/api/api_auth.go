package api

import (
	"net/http"
	"time"

	"github.com/YouWantToPinch/pincher-api/internal/auth"
	"github.com/YouWantToPinch/pincher-api/internal/database"
	"github.com/golang-jwt/jwt/v5"
)

func (cfg *APIConfig) endpLoginUser(w http.ResponseWriter, r *http.Request) {
	type rqSchema struct {
		Password string `json:"password"`
		Username string `json:"username"`
	}

	rqPayload, err := decodePayload[rqSchema](r)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not log in user", err)
		return
	}

	if rqPayload.Username == "" || rqPayload.Password == "" {
		respondWithError(w, http.StatusBadRequest, "Missing credential(s)", nil)
		return
	}

	dbUser, err := cfg.db.GetUserByUsername(r.Context(), rqPayload.Username)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Incorrect username or password", err)
		return
	}

	match, err := auth.CheckPasswordHash(rqPayload.Password, dbUser.HashedPassword)
	if err != nil || !match {
		respondWithError(w, http.StatusUnauthorized, "Incorrect username or password", err)
		return
	}

	refreshToken, err := auth.MakeRefreshToken()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Trouble logging in", err)
		return
	}
	_, err = cfg.db.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
		Token:  refreshToken,
		UserID: dbUser.ID,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Trouble logging in", err)
		return
	}

	accessToken, err := auth.MakeJWT(dbUser.ID, jwt.SigningMethodHS256, cfg.secret, time.Hour)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Trouble logging in", err)
		return
	}

	rspPayload := User{
		ID:           dbUser.ID,
		CreatedAt:    dbUser.CreatedAt,
		UpdatedAt:    dbUser.UpdatedAt,
		Username:     dbUser.Username,
		Token:        accessToken,
		RefreshToken: refreshToken,
	}

	respondWithJSON(w, http.StatusOK, rspPayload)
}

func (cfg *APIConfig) endpCheckRefreshToken(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't find refresh token", err)
		return
	}

	dbUser, err := cfg.db.GetUserByRefreshToken(r.Context(), refreshToken)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't get user for refresh token", err)
		return
	}

	accessToken, err := auth.MakeJWT(dbUser.ID, jwt.SigningMethodHS256, cfg.secret, time.Hour)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate token", err)
		return
	}

	type rspSchema struct {
		NewAccessToken string `json:"token"`
	}

	rspPayload := rspSchema{
		NewAccessToken: accessToken,
	}

	respondWithJSON(w, http.StatusOK, rspPayload)
}

func (cfg *APIConfig) endpRevokeRefreshToken(w http.ResponseWriter, r *http.Request) {
	rTokenString, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, err.Error(), err)
		return
	}

	dbUser, err := cfg.db.GetUserByRefreshToken(r.Context(), rTokenString)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid or missing token", err)
		return
	}

	err = cfg.db.RevokeUserRefreshToken(r.Context(), dbUser.ID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Refresh Token not found", err)
	}

	respondWithCode(w, http.StatusNoContent)
}
