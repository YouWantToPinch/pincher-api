package server

import (
	"log"
	"net/http"
)

func(cfg *apiConfig) endpDeleteAllUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if cfg.platform != "dev" {
		respondWithText(w, 403, "403 Forbidden")
	}

	err := cfg.db.DeleteUsers(r.Context())
	if err != nil {
		log.Print(err)
	}

	respondWithText(w, 200, "Successfully deleted all users.")
}

func(cfg *apiConfig) endpGetAllUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if cfg.platform != "dev" {
		respondWithText(w, 403, "403 Forbidden")
	}

	users, err := cfg.db.GetAllUsers(r.Context())
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Could not find any users", err)
		return
	}

	var respBody []User
	for _, user := range users {
		respBody = append(respBody, User{
			ID:				user.ID,
			CreatedAt:		user.CreatedAt,
			UpdatedAt:		user.UpdatedAt,
			Username:		user.Username,
			HashedPassword:	user.HashedPassword,
		})
	}

	respondWithJSON(w, http.StatusOK, respBody)
	return
}

func(cfg *apiConfig) endpGetTotalUserCount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if cfg.platform != "dev" {
		respondWithText(w, 403, "403 Forbidden")
	}

	count, err := cfg.db.GetUserCount(r.Context())
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Could not find any users", err)
		return
	}

	respondWithJSON(w, http.StatusOK, count)
	return
}