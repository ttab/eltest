package eltest

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/tern/v2/migrate"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

func NewPostgres(t T) *Postgres {
	pg, err := Bootstrap("postgres", &Postgres{})
	Must(t, err, "bootstrap postgres")

	return pg
}

type Postgres struct {
	res *dockertest.Resource
}

func (pg *Postgres) getPostgresURI(user, database string) string {
	return fmt.Sprintf(
		"postgres://%[1]s:%[1]s@%[3]s:5432/%[2]s",
		user, database, pg.res.Container.NetworkSettings.IPAddress)
}

type PGEnvironment struct {
	migrations fs.FS

	PostgresURI string
}

var sanitizeExp = regexp.MustCompile(`[^a-zA-Z0-9]+`)

func (env *PGEnvironment) Migrator(t T, ctx context.Context, conn *pgx.Conn) *migrate.Migrator {
	t.Helper()

	m, err := migrate.NewMigrator(ctx, conn, "schema_vesion")
	Must(t, err, "create migrator")

	err = m.LoadMigrations(env.migrations)
	Must(t, err, "create load migrations")

	return m
}

const pgAdminUser = "eltest"

func (pg *Postgres) Database(
	t T,
	migrations fs.FS,
	runMigrations bool,
) PGEnvironment {
	t.Helper()

	ctx := context.Background()

	adminConn, err := pgx.Connect(ctx,
		pg.getPostgresURI(pgAdminUser, pgAdminUser))
	Must(t, err, "open postgres admin connection")

	defer adminConn.Close(ctx)

	sane := strings.ToLower(sanitizeExp.ReplaceAllString(t.Name(), "_"))

	_, err = adminConn.Exec(ctx, fmt.Sprintf(`
CREATE ROLE %q WITH LOGIN PASSWORD '%s' REPLICATION`,
		sane, sane))
	Must(t, err, "create user")

	_, err = adminConn.Exec(ctx,
		"CREATE DATABASE "+sane+" WITH OWNER "+sane)
	Must(t, err, "create database")

	env := PGEnvironment{
		migrations:  migrations,
		PostgresURI: pg.getPostgresURI(sane, sane),
	}

	conn, err := pgx.Connect(ctx, env.PostgresURI)
	Must(t, err, "open postgres user connection")

	err = conn.Ping(ctx)
	Must(t, err, "ping postgres user connection")

	defer conn.Close(ctx)

	if runMigrations {
		m := env.Migrator(t, ctx, conn)

		err = m.Migrate(ctx)
		Must(t, err, "migrate to current DB schema")
	}

	return env
}

func (pg *Postgres) SetUp(pool *dockertest.Pool) error {
	res, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "15.2",
		Env: []string{
			"POSTGRES_USER=" + pgAdminUser,
			"POSTGRES_PASSWORD=" + pgAdminUser,
		},
		Cmd: []string{
			"-c", "wal_level=logical",
		},
	}, func(hc *docker.HostConfig) {
		hc.AutoRemove = true
	})
	if err != nil {
		return fmt.Errorf("failed to run postgres container: %w", err)
	}

	pg.res = res

	// Make sure that containers don't stick around for more than an hour,
	// even if in-process cleanup fails.
	_ = res.Expire(3600)

	err = pool.Retry(func() error {
		conn, err := pgx.Connect(context.Background(),
			pg.getPostgresURI(pgAdminUser, pgAdminUser))
		if err != nil {
			return fmt.Errorf("failed to create postgres connection: %w", err)
		}

		err = conn.Ping(context.Background())
		if err != nil {
			log.Println(err.Error())

			return fmt.Errorf("failed to ping database: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to connect to postgres: %w", err)
	}

	return nil
}

func (pg *Postgres) Purge(pool *dockertest.Pool) error {
	if pg.res == nil {
		return nil
	}

	err := pool.Purge(pg.res)
	if err != nil {
		return fmt.Errorf(
			"failed to purge postgres container: %w", err)
	}

	return nil
}
