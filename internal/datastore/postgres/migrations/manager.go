package migrations

import (
	"context"
	"embed"
	"io"
	"path"
	"strings"

	"github.com/authzed/spicedb/pkg/migrate"

	"github.com/jackc/pgx/v4"
)

const (
	dir = "migrations"

	createVersionTable = `CREATE TABLE alembic_version (
    		version_num VARCHAR NOT NULL
		);
		INSERT INTO alembic_version (version_num) VALUES ('');`
)

var (
	//go:embed migrations/*
	migrationFS embed.FS

	// DatabaseMigrations implements a migration manager for the Postgres Driver.
	DatabaseMigrations = migrate.NewManager[*AlembicPostgresDriver, *pgx.Conn, pgx.Tx]()

	GoMigrations = map[string]any{
		"set_unique_id": migrate.TxMigrationFunc[pgx.Tx](set_unique_id),
	}
)

func init() {
	version := "init"
	if err := DatabaseMigrations.Register(version, "", nil, initializeVersionTable); err != nil {
		panic("failed to initialize migrations: " + err.Error())
	}

	migrationFiles, err := migrationFS.ReadDir(dir)
	if err != nil {
		panic("failed to load migrations: " + err.Error())
	}
	for _, migration := range migrationFiles {
		fileName := path.Join(dir, migration.Name())
		file, err := migrationFS.Open(fileName)
		if err != nil {
			panic("failed to load migration " + fileName + ": " + err.Error())
		}
		previous := version
		version = strings.TrimSuffix(migration.Name(), ".sql")

		if strings.HasSuffix(migration.Name(), "sql") {
			sqlBytes, err := io.ReadAll(file)
			if err != nil {
				panic("failed to load migration " + fileName + ": " + err.Error())
			}
			if err := DatabaseMigrations.Register(version, previous, func(ctx context.Context, conn *pgx.Conn) error {
				_, err := conn.Exec(ctx, string(sqlBytes))
				return err
			}, nil); err != nil {
				panic("failed to register migration " + fileName + ": " + err.Error())
			}
		} else if strings.HasSuffix(migration.Name(), "sh") {

			name := strings.TrimSuffix(migration.Name(), ".sh")
			name = name[strings.Index(name, "go_")+3:]

			migrationFn, ok := GoMigrations[name]

			if !ok {
				panic("no go migration registered with name: " + name)
			}

			switch f := migrationFn.(type) {
			case migrate.TxMigrationFunc[pgx.Tx]:
				if err := DatabaseMigrations.Register(version, previous, nil, f); err != nil {
					panic("failed to register migration " + fileName + ": " + err.Error())
				}
			case migrate.MigrationFunc[*pgx.Conn]:
				if err := DatabaseMigrations.Register(version, previous, f, nil); err != nil {
					panic("failed to register migration " + fileName + ": " + err.Error())
				}
			default:
				panic("unknown migration type")
			}
		}
	}
}

func initializeVersionTable(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, createVersionTable)
	return err
}
