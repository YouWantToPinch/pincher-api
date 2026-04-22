package api

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
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

		accessToken, err := auth.MakeJWT(dbUser.ID, jwt.SigningMethodHS256, cfg.jwtSecret, time.Hour)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "could not create new access token", err)
			return
		}
		rspPayload := cfg.makeAuthPayload(w, r, &dbUser, accessToken, refreshToken)

		if err := tx.Commit(r.Context()); err != nil {
			respondWithError(w, http.StatusInternalServerError, "", err)
			return
		}
		respondWithJSON(w, http.StatusOK, rspPayload)
	}
}

func (cfg *APIConfig) handleCheckRefreshToken(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := FindRefreshToken(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "", err)
		return
	}

	dbUser, err := cfg.db.GetUserByRefreshToken(r.Context(), refreshToken)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "could not get user by refresh token", err)
		return
	}

	accessToken, err := auth.MakeJWT(dbUser.ID, jwt.SigningMethodHS256, cfg.jwtSecret, time.Hour)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "could not validate access token", err)
		return
	}

	rspPayload := cfg.makeAuthPayload(w, r, &dbUser, accessToken, refreshToken)
	respondWithJSON(w, http.StatusOK, rspPayload)
}

func (cfg *APIConfig) handleRevokeRefreshToken(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := FindRefreshToken(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "", err)
		return
	}
	if r.Header.Get("X-Auth-Transport") == "cookie" {
		noCookie := &http.Cookie{
			Name:     "refresh_token",
			Value:    "",
			MaxAge:   -1,
			HttpOnly: true,
		}
		http.SetCookie(w, noCookie)
	}

	err = cfg.db.RevokeRefreshToken(r.Context(), refreshToken)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not revoke refresh token", err)
	}

	respondWithCode(w, http.StatusNoContent)
}

// makeAuthPayload prepares and returns a response body relevant to requests
// governing management of refresh tokens. Callers may optionally omit the
// refresh token field from the response payload, opting for an HttpOnly cookie
// instead for browser contexts.
func (cfg *APIConfig) makeAuthPayload(w http.ResponseWriter, r *http.Request, dbUser *database.User, accessToken, refreshToken string) any {
	contains := func(sub string) bool { return strings.Contains(r.URL.String(), sub) }

	asksForUser := (contains("/refresh") || contains("/revoke")) && r.URL.Query().Has("with-user")
	returnUser := (asksForUser || contains("/login")) && dbUser != nil
	forBrowser := r.Header.Get("X-Auth-Transport") == "cookie"

	var user *User
	if dbUser != nil {
		slog.Debug("!= nil! We are doing it.")
		user = &User{
			ID:        dbUser.ID,
			CreatedAt: dbUser.CreatedAt,
			UpdatedAt: dbUser.UpdatedAt,
			Username:  dbUser.Username,
		}
	}

	if forBrowser {
		cookie := &http.Cookie{
			Name:     "refresh_token",
			Value:    refreshToken,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		}
		http.SetCookie(w, cookie)
	}

	// returned fields will vary dependent on whether the
	// response is meant for a browser
	if returnUser {
		if forBrowser {
			return struct {
				User
				Token string `json:"token"`
			}{
				User:  *user,
				Token: accessToken,
			}
		} else {
			return struct {
				User
				Token        string `json:"token"`
				RefreshToken string `json:"refresh_token"`
			}{
				User:         *user,
				Token:        accessToken,
				RefreshToken: refreshToken,
			}
		}
	} else {
		return struct {
			Token string `json:"token"`
		}{
			Token: accessToken,
		}
	}
}

// FindRefreshToken attempts to find a refresh token in
// the Authorization header, unless the X-Auth-Transport
// header indicates that it should search in an existing
// refresh_token cookie.
func FindRefreshToken(r *http.Request) (string, error) {
	var refreshToken string
	var err error
	if r.Header.Get("X-Auth-Transport") != "cookie" {
		refreshToken, err = auth.GetBearerToken(r.Header)
		if err != nil {
			return "", fmt.Errorf("could not get refresh token from Authorization header: %w", err)
		}
	} else {
		refreshToken, err = auth.GetRefreshTokenFromCookie(r)
		if err != nil {
			return "", fmt.Errorf("could not get refresh token from refresh_token cookie: %w", err)
		}
	}
	return refreshToken, err
}
