package main

import (
	"encoding/json"
	"net/http"
	"time"
	"fmt"

	"github.com/YouWantToPinch/pincher-api/internal/auth"
	"github.com/YouWantToPinch/pincher-api/internal/database"
)

func(cfg *apiConfig) endpLoginUser(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Password			string	`json:"password"`
		Username			string	`json:"username`
	}

	decoder := json.NewDecoder(r.Body)
    params := parameters{}
    err := decoder.Decode(&params)
    if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not log in user", err)
		return
    }

	dbUser, err := cfg.db.GetUserByUsername(r.Context(), params.Username)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Incorrect email or password", err)
		return
	}

	err = auth.CheckPasswordHash(params.Password, dbUser.HashedPassword)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Incorrect email or password", err)
		return
	}

	refreshToken, err := auth.MakeRefreshToken()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Trouble logging in", err)
		return
	}
	_, err = cfg.db.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
		Token:	refreshToken,
		UserID:	dbUser.ID,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Trouble logging in", err)
		return
	}

	accessToken, err := auth.MakeJWT(dbUser.ID, cfg.secret, time.Hour)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Trouble logging in", err)
		return
	}
	
	respBody := User{
		ID:        		dbUser.ID,
		CreatedAt: 		dbUser.CreatedAt,
		UpdatedAt: 		dbUser.UpdatedAt,
		Username:     	dbUser.Username,
		Token:			accessToken,
		RefreshToken:	refreshToken,
	}

	respondWithJSON(w, http.StatusOK, respBody)
	return
}

func(cfg *apiConfig) endpCheckRefreshToken(w http.ResponseWriter, r *http.Request) {
	type returnVals struct {
		NewAccessToken string `json:"token"`
	}
	
	rTokenString, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, err.Error(), err)
		return
	}

	dbRefreshToken, err := cfg.db.GetRefreshToken(r.Context(), rTokenString)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, err.Error(), err)
		return
	} else if dbRefreshToken.ExpiresAt.Before(time.Now()) || dbRefreshToken.RevokedAt.Valid == true {
		respondWithError(w, http.StatusUnauthorized, "Invalid or missing token", nil)
		return
	}

	dbUser, err := cfg.db.GetUserByRefreshToken(r.Context(), rTokenString)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid or missing token", err)
		return
	}

	newJWTToken, err := auth.MakeJWT(dbUser.ID, cfg.secret, time.Hour)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error(), err)
		return
	}

	respBody := returnVals{
		NewAccessToken: newJWTToken,
	}

	respondWithJSON(w, http.StatusOK, respBody)
	return
}

func(cfg *apiConfig) endpRevokeRefreshToken(w http.ResponseWriter, r *http.Request) {
	
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
	
	cfg.db.RevokeUserRefreshToken(r.Context(), dbUser.ID)

	respMsg := fmt.Sprintf("Revoked refresh token for user: %s", dbUser.Username)
	respondWithText(w, http.StatusNoContent, respMsg)
	return
}