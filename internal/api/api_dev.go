package api

import (
	"log/slog"
	"net/http"
)

func (cfg *apiConfig) endpDeleteAllUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if cfg.platform != "dev" {
		respondWithText(w, 403, "403 Forbidden")
	}

	err := cfg.db.DeleteUsers(r.Context())
	if err != nil {
		slog.Error(err.Error())
	}

	respondWithText(w, 200, "Successfully deleted all users.")
}

func (cfg *apiConfig) endpGetAllUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if cfg.platform != "dev" {
		respondWithText(w, 403, "403 Forbidden")
	}

	dbUsers, err := cfg.db.GetAllUsers(r.Context())
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Could not find any users", err)
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

	type resp struct {
		Users []User `json:"users"`
	}

	respBody := resp{
		Users: users,
	}

	respondWithJSON(w, http.StatusOK, respBody)
}

func (cfg *apiConfig) endpGetTotalUserCount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if cfg.platform != "dev" {
		respondWithText(w, 403, "403 Forbidden")
	}

	count, err := cfg.db.GetUserCount(r.Context())
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Could not find any users", err)
		return
	}

	type resp struct {
		Count int64 `json:"count"`
	}

	respBody := resp{
		Count: count,
	}

	respondWithJSON(w, http.StatusOK, respBody)
}
