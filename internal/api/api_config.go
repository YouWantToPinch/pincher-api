package api

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"

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
