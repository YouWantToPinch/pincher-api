package api

import (
	"net/http"
	"time"

	"github.com/YouWantToPinch/pincher-api/internal/auth"
	"github.com/YouWantToPinch/pincher-api/internal/database"
	"github.com/golang-jwt/jwt/v5"
)

func (cfg *APIConfig) handleLoginUser(w http.ResponseWriter, r *http.Request) {
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
		respondWithError(w, http.StatusBadRequest, "missing credential(s)", nil)
		return
	}

	dbUser, err := cfg.db.GetUserByUsername(r.Context(), rqPayload.Username)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "incorrect username or password", err)
		return
	}

	match, err := auth.CheckPasswordHash(rqPayload.Password, dbUser.HashedPassword)
	if err != nil || !match {
		respondWithError(w, http.StatusUnauthorized, "incorrect username or password", err)
		return
	}

	refreshToken, err := auth.MakeRefreshToken()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not create refresh token", err)
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

		_, err = q.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
			Token:     refreshToken,
			UserID:    dbUser.ID,
			ExpiresAt: time.Now().UTC().UTC().Add(time.Hour * 24 * 30),
		})
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "could not save refresh token", err)
			return
		}

		accessToken, err := auth.MakeJWT(dbUser.ID, jwt.SigningMethodHS256, cfg.secret, time.Hour)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "could not create new access token", err)
			return
		}

		type rspSchema struct {
			User
			Token        string `json:"token"`
			RefreshToken string `json:"refresh_token"`
		}

		rspPayload := rspSchema{
			User: User{
				ID:        dbUser.ID,
				CreatedAt: dbUser.CreatedAt,
				UpdatedAt: dbUser.UpdatedAt,
				Username:  dbUser.Username,
			},
			Token:        accessToken,
			RefreshToken: refreshToken,
		}
		if err := tx.Commit(r.Context()); err != nil {
			respondWithError(w, http.StatusInternalServerError, "", err)
			return
		}
		respondWithJSON(w, http.StatusOK, rspPayload)
	}
}

func (cfg *APIConfig) handleCheckRefreshToken(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "could not get refresh token from header", err)
		return
	}

	dbUser, err := cfg.db.GetUserByRefreshToken(r.Context(), refreshToken)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "could not get user by refresh token", err)
		return
	}

	accessToken, err := auth.MakeJWT(dbUser.ID, jwt.SigningMethodHS256, cfg.secret, time.Hour)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "could not validate access token", err)
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

func (cfg *APIConfig) handleRevokeRefreshToken(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "could not get refresh token", err)
		return
	}

	err = cfg.db.RevokeRefreshToken(r.Context(), refreshToken)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not revoke refresh token", err)
	}

	respondWithCode(w, http.StatusNoContent)
}
