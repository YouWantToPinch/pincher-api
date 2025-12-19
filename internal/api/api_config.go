package api

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"

	"github.com/YouWantToPinch/pincher-api/internal/auth"
	"github.com/YouWantToPinch/pincher-api/internal/database"
)

type APIConfig struct {
	db       *database.Queries
	dbURL    string
	platform string
	secret   string
	logger   *slog.Logger
	// apiKeys  *map[string]string
}

func (cfg *APIConfig) Init(envPath string, altDBUrl string) {
	// get environment variables
	if len(envPath) != 0 {
		_ = godotenv.Load(envPath)
	}

	cfg.platform = os.Getenv("PLATFORM")
	cfg.secret = os.Getenv("SECRET")

	if len(altDBUrl) != 0 {
		cfg.dbURL = altDBUrl
	} else {
		cfg.GenerateDBConnectionString()
	}

	{
		slogLevel := os.Getenv("SLOG_LEVEL")
		switch slogLevel {
		case "DEBUG":
			cfg.NewLogger(slog.LevelDebug)
		case "WARN":
			cfg.NewLogger(slog.LevelWarn)
		case "ERROR":
			cfg.NewLogger(slog.LevelError)
		default:
			cfg.NewLogger(slog.LevelInfo)
		}
	}
}

func (cfg *APIConfig) NewLogger(level slog.Level) {
	cfg.logger = slog.New(slog.NewJSONHandler(os.Stdout,
		&slog.HandlerOptions{Level: level}))
	slog.SetDefault(cfg.logger)
}

func (cfg *APIConfig) GenerateDBConnectionString() *string {
	envOrDefault := func(envVar string, defaultVal string) string {
		envVal := os.Getenv(envVar)
		if len(envVal) == 0 {
			envVal = defaultVal
		}
		return envVal
	}

	dbUser := envOrDefault("DB_USER", "postgres")
	dbPassword := envOrDefault("DB_PASSWORD", "postgres")
	dbHost := envOrDefault("DB_HOST", "localhost")
	dbPort := envOrDefault("DB_PORT", "5432")
	dbName := envOrDefault("DB_NAME", "pincher")

	cfg.dbURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		dbUser,
		dbPassword,
		dbHost,
		dbPort,
		dbName,
	)
	return &cfg.dbURL
}

func (cfg *APIConfig) ConnectToDB(fs embed.FS, migrationsDir string) {
	db, err := sql.Open("postgres", cfg.dbURL)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Default to relative directory so tests know where to find migrations
	// Otherwise, use embedded directory in a compiled binary context
	if len(migrationsDir) == 0 {
		migrationsDir = "../../sql/schema"
	} else {
		goose.SetBaseFS(fs)
	}

	if err := goose.SetDialect("postgres"); err != nil {
		panic(err)
	}

	if err = goose.Up(db, migrationsDir); err != nil {
		slog.Error("could not apply database migrations with goose; " + err.Error())
		panic(err)
	}

	cfg.db = database.New(db)
}

// ================= MIDDLEWARE ================= //

type ctxKey string

func (cfg *APIConfig) middlewareAuthenticate(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString, err := auth.GetBearerToken(r.Header)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, err.Error(), err)
			slog.Error("Couldn't get bearer token")
			return
		}
		validatedUserID, err := auth.ValidateJWT(tokenString, cfg.secret, "HS256")
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

func (cfg *APIConfig) middlewareCheckClearance(required BudgetMemberRole, next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		validatedUserID := getContextKeyValue(r.Context(), "user_id")

		var pathBudgetID uuid.UUID
		err := parseUUIDFromPath("budget_id", r, &pathBudgetID)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid parameter value", err)
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
