package api

import (
	"net/http"
)

func (cfg *APIConfig) handleDeleteAllUsers(w http.ResponseWriter, r *http.Request) {
	if cfg.platform != "dev" {
		respondWithText(w, http.StatusForbidden, "platform not dev")
	}

	err := cfg.db.DeleteUsers(r.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not delete users", err)
	}

	respondWithCode(w, http.StatusNoContent)
}

func (cfg *APIConfig) handleGetAllUsers(w http.ResponseWriter, r *http.Request) {
	if cfg.platform != "dev" {
		respondWithText(w, http.StatusForbidden, "platform not dev")
	}

	dbUsers, err := cfg.db.GetAllUsers(r.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not retrieve users", err)
		return
	}

	users := []User{}
	for _, dbUser := range dbUsers {
		users = append(users, User{
			ID:             dbUser.ID,
			CreatedAt:      dbUser.CreatedAt,
			UpdatedAt:      dbUser.UpdatedAt,
			Username:       dbUser.Username,
			HashedPassword: dbUser.HashedPassword,
		})
	}

	type rspSchema struct {
		Users []User `json:"users"`
	}

	rspPayload := rspSchema{
		Users: users,
	}

	respondWithJSON(w, http.StatusOK, rspPayload)
}

func (cfg *APIConfig) handleGetTotalUserCount(w http.ResponseWriter, r *http.Request) {
	if cfg.platform != "dev" {
		respondWithText(w, http.StatusForbidden, "platform not dev")
	}

	count, err := cfg.db.GetUserCount(r.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not calculate user count", err)
		return
	}

	type rspSchema struct {
		Count int64 `json:"count"`
	}

	rspPayload := rspSchema{
		Count: count,
	}

	respondWithJSON(w, http.StatusOK, rspPayload)
}
