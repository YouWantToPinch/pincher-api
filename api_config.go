package main

import (
	"log"
	"context"

	"sync/atomic"
	"net/http"

	"github.com/google/uuid"

	"github.com/YouWantToPinch/pincher-api/internal/auth"
	"github.com/YouWantToPinch/pincher-api/internal/database"
)

type apiConfig struct {
	// atomic.Int32 is a //standard-library type that allows us to 
	// safely increment and read an integer value across multiple 
	// goroutines (HTTP requests)
	fileserverHits	atomic.Int32
	db				*database.Queries
	platform		string
	secret			string
	apiKeys			*map[string]string
}

// ================= MIDDLEWARE ================= //
func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) middlewareMetricsReset(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Store(0)
		next.ServeHTTP(w, r)
	})
}

type ctxKey struct{}
var ctxUserID ctxKey

func (cfg *apiConfig) middlewareAuthenticate(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString, err := auth.GetBearerToken(r.Header)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, err.Error(), err)
			return
		}
		validatedUserID, err := auth.ValidateJWT(tokenString, cfg.secret)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "401 Unauthorized", nil)
			return
		}
		ctx := context.WithValue(r.Context(), ctxUserID, validatedUserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ============== HELPERS =================

func getValidatedUserID(ctx context.Context) (uuid.UUID) {
    validatedUserID, ok := ctx.Value(ctxUserID).(uuid.UUID)
	if !ok {
		log.Println("Failed to retrieve validated user_id from context")
		return uuid.Nil
	}
    return validatedUserID
}