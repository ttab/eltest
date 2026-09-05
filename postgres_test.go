package eltest_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/ttab/eltest"
)

func TestPostgres(t *testing.T) {
	versions := []string{
		eltest.Postgres17_6,
		eltest.Postgres18_6,
	}

	for _, tag := range versions {
		t.Run(tag, func(t *testing.T) {
			testPostgresVersion(t, tag)
		})
	}
}

func testPostgresVersion(t *testing.T, tag string) {
	t.Helper()

	pg := eltest.NewPostgres(t, tag)

	migrationFS := os.DirFS(filepath.Join("testdata", "migrations"))

	t.Run("WithMigrations", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		t.Cleanup(cancel)

		pgEnv := pg.Database(t, "example", migrationFS, true)

		conn, err := pgx.Connect(ctx, pgEnv.PostgresURI)
		eltest.Must(t, err, "connect to database")

		_, err = conn.Exec(ctx,
			`INSERT INTO example(name, description) VALUES('hello', 'world')`)
		eltest.Must(t, err, "write to database")

		row := conn.QueryRow(ctx,
			`SELECT description FROM example WHERE name = 'hello'`)

		var desc string

		err = row.Scan(&desc)
		eltest.Must(t, err, "read description from db")

		if desc != testValue {
			t.Fatalf("got %q back, expected %q", desc, testValue)
		}
	})

	t.Run("ManualMigrations", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		t.Cleanup(cancel)

		pgEnv := pg.Database(t, "example", migrationFS, false)

		conn, err := pgx.Connect(ctx, pgEnv.PostgresURI)
		eltest.Must(t, err, "connect to database")

		migrator := pgEnv.Migrator(ctx, t, conn)

		err = migrator.MigrateTo(ctx, 1)
		eltest.Must(t, err, "migrate to schema v1")

		_, err = conn.Exec(ctx,
			`INSERT INTO example(name) VALUES('elephant')`)
		eltest.Must(t, err, "write a name to the v1 database")

		_, err = conn.Exec(ctx,
			`INSERT INTO example(name, description) VALUES('hello', 'world')`)
		if err == nil {
			t.Fatalf("should not be able to insert description in a v1 database")
		}

		err = migrator.MigrateTo(ctx, 2)
		eltest.Must(t, err, "migrate to schema v2")

		_, err = conn.Exec(ctx,
			`INSERT INTO example(name, description) VALUES('hello', 'world')`)
		eltest.Must(t, err, "write name and description to database")
	})
}

func TestPostgresConcurrent(t *testing.T) {
	migrationFS := os.DirFS(filepath.Join("testdata", "migrations"))

	for i := range 10 {
		t.Run(fmt.Sprintf("N%d", i), func(t *testing.T) {
			t.Parallel()

			pg := eltest.NewPostgres(t, eltest.Postgres17_6)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			t.Cleanup(cancel)

			pgEnv := pg.Database(t, "example", migrationFS, true)

			conn, err := pgx.Connect(ctx, pgEnv.PostgresURI)
			eltest.Must(t, err, "connect to database")

			_, err = conn.Exec(ctx,
				`INSERT INTO example(name, description) VALUES('hello', 'world')`)
			eltest.Must(t, err, "write to database")
		})
	}
}
