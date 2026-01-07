package api

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"

	"github.com/YouWantToPinch/pincher-api/internal/auth"
	"github.com/YouWantToPinch/pincher-api/internal/database"
)

type APIConfig struct {
	db       *database.Queries
	Pool     *pgxpool.Pool
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

	// Create a temporary *sql.DB so that goose can apply migrations
	pgxConfig, err := pgx.ParseConfig(cfg.dbURL)
	if err != nil {
		slog.Error("could not apply database migrations with goose: " + err.Error())
		panic(err)
	}
	sqlDB := stdlib.OpenDB(*pgxConfig)

	if err := goose.Up(sqlDB, migrationsDir); err != nil {
		slog.Error("could not apply database migrations with goose " + err.Error())
		panic(err)
	} else {
		err := sqlDB.Close()
		if err != nil {
			panic(err)
		}
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.dbURL)
	if err != nil {
		slog.Error("could not connect to postgres database: %w" + err.Error())
		panic(err)
	}
	cfg.Pool = pool

	cfg.db = database.New(pool)
}

// ================= MIDDLEWARE ================= //

type ctxKey string

func (cfg *APIConfig) middlewareAuthenticate(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString, err := auth.GetBearerToken(r.Header)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "no token found", err)
			return
		}
		validatedUserID, err := auth.ValidateJWT(tokenString, cfg.secret, "HS256")
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "invalid token provided", nil)
			return
		}
		ctxUserID := ctxKey("user_id")
		ctx := context.WithValue(r.Context(), ctxUserID, validatedUserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (cfg *APIConfig) middlewareCheckClearance(required BudgetMemberRole, next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		validatedUserID := getContextKeyValueAsUUID(r.Context(), "user_id")

		var pathBudgetID uuid.UUID
		err := parseUUIDFromPath("budget_id", r, &pathBudgetID)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "", err)
			return
		}

		callerRole, err := cfg.db.GetBudgetMemberRole(r.Context(), database.GetBudgetMemberRoleParams{
			BudgetID: pathBudgetID,
			UserID:   validatedUserID,
		})
		if err != nil {
			respondWithError(w, http.StatusForbidden, "user not found as member", err)
			return
		}

		callerBudgetMemberRole, err := BMRFromString(callerRole)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "invalid role", err)
			return
		}

		if callerBudgetMemberRole > required {
			respondWithError(w, http.StatusForbidden, "user does not have clearance for action", err)
			return
		}
		ctxBudgetID := ctxKey("budget_id")
		ctx := context.WithValue(r.Context(), ctxBudgetID, pathBudgetID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// middlewareValidateTxn validates transaction request payloads,
// then converts relevant resource names to their corresponding UUIDs where valid,
// in preparation for the database query to log the transaction.
func (cfg *APIConfig) middlewareValidateTxn(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pathBudgetID := getContextKeyValueAsUUID(r.Context(), "budget_id")
		rqPayload, err := decodePayload[UpsertTransactionRqSchema](r)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "", err)
			return
		}
		validatedTxn, err := validateTxnInput(&rqPayload)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "", err)
			return
		}
		if validatedTxn.txnType == "NONE" {
			respondWithError(w, http.StatusInternalServerError, "transaction type could not be inferred", nil)
		}

		accountID, err := lookupResourceIDByName(r.Context(),
			database.GetBudgetAccountIDByNameParams{
				AccountName: rqPayload.AccountName,
				BudgetID:    pathBudgetID,
			}, cfg.db.GetBudgetAccountIDByName)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "could not get account id", err)
			return
		}
		validatedTxn.accountID = *accountID

		if validatedTxn.isTransfer {
			transferAccountID, err := lookupResourceIDByName(r.Context(),
				database.GetBudgetAccountIDByNameParams{
					AccountName: rqPayload.TransferAccountName,
					BudgetID:    pathBudgetID,
				}, cfg.db.GetBudgetAccountIDByName)
			if err != nil {
				respondWithError(w, http.StatusBadRequest, "could not get transfer account id", err)
				return
			}
			validatedTxn.transferAccountID = *transferAccountID
		} else {
			payeeID, err := lookupResourceIDByName(r.Context(),
				database.GetBudgetPayeeIDByNameParams{
					PayeeName: rqPayload.PayeeName,
					BudgetID:  pathBudgetID,
				}, cfg.db.GetBudgetPayeeIDByName)
			if err != nil {
				respondWithError(w, http.StatusBadRequest, "could not get payee id", err)
				return
			}
			validatedTxn.payeeID = *payeeID
		}

		// convert names to IDs if needed
		for k, v := range rqPayload.Amounts {
			if _, ok := validatedTxn.amounts[k]; !ok {
				slog.Info("SKIPPING ELIMINATED KEY: " + k)
				// validation already weeded this one out; move on to the next
				continue
			}
			if k == "TRANSFER" || (k == "UNCATEGORIZED" && validatedTxn.txnType == "DEPOSIT") {
				slog.Info("SKIPPING IRRELEVANT KEY: " + k)
				// categories are not relevant
				continue
			}
			slog.Info(fmt.Sprintf("GETTING UUID FOR KEY: %s, AMOUNT: %d", k, v))
			categoryID, err := lookupResourceIDByName(r.Context(),
				database.GetBudgetCategoryIDByNameParams{
					CategoryName: k,
					BudgetID:     pathBudgetID,
				}, cfg.db.GetBudgetCategoryIDByName)
			if err != nil {
				var errMessage string
				if len(rqPayload.Amounts) > 1 {
					errMessage = "could not get category id for one or more transaction splits"
				} else {
					errMessage = "could not get category id for transaction"
				}
				respondWithError(w, http.StatusBadRequest, errMessage, err)
				return
			}
			validatedTxn.amounts[categoryID.String()] = v
			delete(validatedTxn.amounts, k)
		}

		ctxValidatedTxn := ctxKey("validated_txn")
		ctx := context.WithValue(r.Context(), ctxValidatedTxn, validatedTxn)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ============== HELPERS =================

func getContextKeyValueAsUUID(ctx context.Context, key string) uuid.UUID {
	contextKeyValue, ok := ctx.Value(ctxKey(key)).(uuid.UUID)
	if !ok {
		slog.Warn("failed to retrieve key from context", slog.String("key", key))
		return uuid.Nil
	}
	return contextKeyValue
}

func getContextKeyValueAsTxn(ctx context.Context, key string) *validatedTxnPayload {
	contextKeyValue, ok := ctx.Value(ctxKey(key)).(*validatedTxnPayload)
	if !ok {
		slog.Warn("failed to retrieve key from context", slog.String("key", key))
		return nil
	}
	return contextKeyValue
}

// validateTxnInput parses relevant inputs: txn amounts, txnDate, transfer status, txnType.
// Any txn with no amount, or with amounts not matching in type, are rejected.
// Any error returned implies a bad request.
func validateTxnInput(rqPayload *UpsertTransactionRqSchema) (*validatedTxnPayload, error) {
	validatedTxn := &validatedTxnPayload{amounts: map[string]int64{}}
	var err error

	validatedTxn.txnDate, err = time.Parse("2006-01-02", rqPayload.TransactionDate)
	if err != nil {
		return nil, fmt.Errorf("transaction date could not be parsed")
	}

	validatedTxn.isTransfer = (rqPayload.TransferAccountName != "")
	validatedTxn.txnType = "NONE"

	setTxnType := func(ptr *string, val string) error {
		switch *ptr {
		case "NONE":
			*ptr = val
			return nil
		case val:
			return nil
		default:
			return fmt.Errorf("one or more splits do not match expected type '%v'", *ptr)
		}
	}

	maps.Copy(validatedTxn.amounts, rqPayload.Amounts)
	for k, v := range rqPayload.Amounts {
		if k == "" {
			return nil, fmt.Errorf("found missing category name from one or more amount fields")
		}
		switch {
		case v > 0:
			if validatedTxn.isTransfer {
				err = setTxnType(&validatedTxn.txnType, "TRANSFER_TO")
			} else {
				err = setTxnType(&validatedTxn.txnType, "DEPOSIT")
			}
		case v < 0:
			if validatedTxn.isTransfer {
				err = setTxnType(&validatedTxn.txnType, "TRANSFER_FROM")
			} else {
				err = setTxnType(&validatedTxn.txnType, "WITHDRAWAL")
			}
		default:
			delete(validatedTxn.amounts, k)
		}
		// return error on txnType mismatch
		if err != nil {
			return nil, fmt.Errorf("inconsistent signage on amount values")
		}
	}
	// return error on txn amount of 0
	if len(validatedTxn.amounts) == 0 {
		return nil, fmt.Errorf("no non-zero amount specified for transaction")
	}
	return validatedTxn, nil
}
