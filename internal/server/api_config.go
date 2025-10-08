package server

import (
	"context"
	"log/slog"
	"os"

	"net/http"
	"sync/atomic"

	"github.com/google/uuid"

	"github.com/YouWantToPinch/pincher-api/internal/auth"
	"github.com/YouWantToPinch/pincher-api/internal/database"
)

type apiConfig struct {
	// atomic.Int32 is a //standard-library type that allows us to
	// safely increment and read an integer value across multiple
	// goroutines (HTTP requests)
	fileserverHits atomic.Int32
	db             *database.Queries
	platform       string
	secret         string
	logger         *slog.Logger
	apiKeys        *map[string]string
}

func (cfg *apiConfig) Init(level slog.Level) {
	cfg.logger = slog.New(slog.NewJSONHandler(os.Stdout,
		&slog.HandlerOptions{Level: level}))
	slog.SetDefault(cfg.logger)
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

type ctxKey string

func (cfg *apiConfig) middlewareAuthenticate(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString, err := auth.GetBearerToken(r.Header)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, err.Error(), err)
			slog.Error("Couldn't get bearer token")
			return
		}
		validatedUserID, err := auth.ValidateJWT(tokenString, cfg.secret)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "401 Unauthorized", nil)
			slog.Error("Failed validation for JWT: " + tokenString)
			return
		}
		ctxUserID := ctxKey("user_id")
		ctx := context.WithValue(r.Context(), ctxUserID, validatedUserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (cfg *apiConfig) middlewareCheckClearance(required BudgetMemberRole, next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		validatedUserID := getContextKeyValue(r.Context(), "user_id")

		idString := r.PathValue("budget_id")
		pathBudgetID, err := uuid.Parse(idString)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Something went wrong", err)
			slog.Debug("Could not parse budget_id")
			return
		}

		callerRole, err := cfg.db.GetBudgetMemberRole(r.Context(), database.GetBudgetMemberRoleParams{
			BudgetID: pathBudgetID,
			UserID:   validatedUserID,
		})
		if err != nil {
			respondWithError(w, http.StatusNotFound, "User not listed as budget member", err)
			return
		}

		callerBudgetMemberRole, err := BMRFromString(callerRole)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error(), err)
			return
		}

		if callerBudgetMemberRole > required {
			respondWithError(w, http.StatusUnauthorized, "Member does not have clearance for action", err)
			return
		}
		ctxBudgetID := ctxKey("budget_id")
		ctx := context.WithValue(r.Context(), ctxBudgetID, pathBudgetID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ============== HELPERS =================

func getContextKeyValue(ctx context.Context, key string) uuid.UUID {
	contextKeyValue, ok := ctx.Value(ctxKey(key)).(uuid.UUID)
	if !ok {
		slog.Info("Failed to retrieve key from context")
		return uuid.Nil
	}
	return contextKeyValue
}
