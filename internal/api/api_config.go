package api

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"

	"github.com/YouWantToPinch/pincher-api/internal/database"
)

type DBConfig struct {
	DBUser     string
	DBPassword string
	DBHost     string
	DBPort     string
	DBName     string
}

type APIConfig struct {
	db        *database.Queries
	Pool      *pgxpool.Pool
	dbURL     string
	platform  string
	jwtSecret string
	logger    *slog.Logger
	// apiKeys  *map[string]string
}

func (cfg *APIConfig) Init(envPath string) error {
	// get environment variables from .env where not set,
	// if platform is set to dev
	if len(envPath) != 0 {
		_ = godotenv.Load(envPath)
	}

	cfg.platform = os.Getenv("PLATFORM")

	cfg.jwtSecret = os.Getenv("JWT_SECRET")

	altDBUrl := os.Getenv("DB_URL")
	if len(altDBUrl) > 0 {
		cfg.dbURL = altDBUrl
	} else {
		var err error
		cfg.dbURL, err = cfg.GenerateDBConnectionString()
		if err != nil {
			return err
		}
	}

	{
		slogLevel := os.Getenv("SLOG_LEVEL")
		switch slogLevel {
		case "DEBUG":
			cfg.NewLogger(slog.LevelDebug)
		case "INFO":
			cfg.NewLogger(slog.LevelInfo)
		case "WARN":
			cfg.NewLogger(slog.LevelWarn)
		case "ERROR":
			cfg.NewLogger(slog.LevelError)
		default:
			cfg.NewLogger(slog.LevelInfo)
		}
	}
	return nil
}

func (cfg *APIConfig) NewLogger(level slog.Level) {
	cfg.logger = slog.New(slog.NewJSONHandler(os.Stdout,
		&slog.HandlerOptions{Level: level}))
	slog.SetDefault(cfg.logger)
}

func (cfg *APIConfig) GenerateDBConnectionString() (string, error) {
	const (
		USER = "DB_USER"
		PSWD = "DB_PASSWORD"
		HOST = "DB_HOST"
		PORT = "DB_PORT"
		NAME = "DB_NAME"
		SSLM = "DB_SSLMODE"
	)
	dbURLMap := map[string]string{
		USER: os.Getenv(USER), // postgres
		PSWD: os.Getenv(PSWD), // postgres
		HOST: os.Getenv(HOST), // localhost
		PORT: os.Getenv(PORT), // 5432
		NAME: os.Getenv(NAME), // pincher
		SSLM: os.Getenv(SSLM), // pincher
	}

	var missing []string
	for k, v := range dbURLMap {
		if v == "" {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		return "", fmt.Errorf("missing environment variables: %s", strings.Join(missing, ", "))
	}

	url := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		dbURLMap[USER],
		dbURLMap[PSWD],
		dbURLMap[HOST],
		dbURLMap[PORT],
		dbURLMap[NAME],
		dbURLMap[SSLM],
	)

	/*
		postgres://DB_USER:DB_PASSWORD@DB_HOST:DB_PORT/DB_NAME?sslmode=disable"
		postgres://postgres:postgres@localhost:5432/pincher?sslmode=disable"
	*/

	return url, nil
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

	// TODO: When documentation can recommend a more graceful and
	// conventional method of migration, limit this feature to
	// ONLY be available while PLATFORM == 'dev'.
	//
	// if pl := cfg.platform; pl == "dev" || pl == "test" {}

	migrate := os.Getenv("MIGRATE_ON_START")
	if migrate == "true" {
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
