package main

import (
	"log"
	"net/http"
	"encoding/json"

	"github.com/YouWantToPinch/pincher-api/internal/database"
	"github.com/YouWantToPinch/pincher-api/internal/auth"
)

func(cfg *apiConfig) endpCreateUser(w http.ResponseWriter, r *http.Request){
    type parameters struct {
		Password	string	`json:"password"`
        Username	string	`json:"username"`
    }

    decoder := json.NewDecoder(r.Body)
    params := parameters{}
    err := decoder.Decode(&params)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Failure decoding parameters", err)
		return
    }

	if params.Username == "" || params.Password == "" {
		respondWithError(w, http.StatusBadRequest, "Missing username or password", nil)
		return
	}

	hashedPass, err := auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failure processing request to create user", err)
		return
	}

	dbUser, err := cfg.db.CreateUser(r.Context(), database.CreateUserParams{
		Username:		params.Username,
		HashedPassword:	hashedPass,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failure processing request to create user", err)
		return
	}
	respBody := User{
		ID:        		dbUser.ID,
		CreatedAt: 		dbUser.CreatedAt,
		UpdatedAt: 		dbUser.UpdatedAt,
		Username:     	dbUser.Username,
	}

	respondWithJSON(w, http.StatusCreated, respBody)
	return
}

func(cfg *apiConfig) endpUpdateUserCredentials(w http.ResponseWriter, r *http.Request){
    type parameters struct {
		Password	string	`json:"password"`
        Username	string	`json:"username"`
    }

	decoder := json.NewDecoder(r.Body)
    params := parameters{}
    err := decoder.Decode(&params)
    if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error decoding parameters", err)
		return
    }

	if params.Username == "" || params.Password == "" {
		respondWithError(w, http.StatusBadRequest, "Missing username or password", nil)
		return
	}

	hashedPass, err := auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failure processing request to create user", err)
		return
	}

	tokenString, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, err.Error(), err)
		return
	}

	dbUserID, err := auth.ValidateJWT(tokenString, cfg.secret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid or missing token", err)
		return
	}

	dbUserUpdated, err := cfg.db.UpdateUserCredentials(r.Context(), database.UpdateUserCredentialsParams{
		ID:					dbUserID,
		Username:				params.Username,
		HashedPassword:		hashedPass,
	})

	respBody := User{
		ID:        		dbUserUpdated.ID,
		CreatedAt: 		dbUserUpdated.CreatedAt,
		UpdatedAt: 		dbUserUpdated.UpdatedAt,
		Username:     	dbUserUpdated.Username,
	}

	respondWithJSON(w, http.StatusOK, respBody)
	return
}

func(cfg *apiConfig) endpDeleteUser(w http.ResponseWriter, r *http.Request){
    type parameters struct {
		Password	string	`json:"password"`
        Username	string	`json:"username"`
    }

	decoder := json.NewDecoder(r.Body)
    params := parameters{}
    err := decoder.Decode(&params)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Error decoding parameters", err)
		return
    }

	if params.Username == "" || params.Password == "" {
		respondWithError(w, http.StatusBadRequest, "Missing username or password", nil)
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

	tokenString, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, err.Error(), err)
		return
	}

	dbUserID, err := auth.ValidateJWT(tokenString, cfg.secret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid or missing token", err)
		return
	}

	cfg.db.DeleteUserByID(r.Context(), dbUserID)
	respondWithText(w, http.StatusOK, "The user was deleted.")
	return
}