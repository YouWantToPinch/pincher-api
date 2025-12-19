package api

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

type postgresContainer struct {
	Ctx       context.Context
	Container postgres.PostgresContainer
	URI       string
}

type StdoutLogConsumer struct{}

func (lc *StdoutLogConsumer) Accept(l tc.Log) {
	if l.LogType == "STDERR" {
		_, err := fmt.Fprintln(os.Stdout, string(l.Content))
		if err != nil {
			fmt.Println("Error writing to stdout:", err)
			return
		}
	}
}

func SetupPostgres(t testing.TB) *postgresContainer {
	t.Helper()
	ctx := context.Background()

	// Ensure migration files exist
	_, err := filepath.Glob("../../sql/schema/*.sql")
	require.NoError(t, err)

	g := StdoutLogConsumer{}

	pgc, err := postgres.Run(
		ctx,
		"postgres:18.1-alpine",
		postgres.WithDatabase("pincher"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		tc.WithLogConsumerConfig(&tc.LogConsumerConfig{
			Consumers: []tc.LogConsumer{&g},
		}),
		postgres.BasicWaitStrategies(),
		tc.WithReuseByName("pinchdb-integration-tests"),
	)
	defer tc.CleanupContainer(t, pgc)
	require.NoError(t, err)

	err = pgc.Snapshot(ctx)
	require.NoError(t, err)

	dbURL, err := pgc.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	return &postgresContainer{Ctx: ctx, Container: *pgc, URI: dbURL}
}
