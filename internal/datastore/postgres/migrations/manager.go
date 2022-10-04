package migrations

import (
	"context"
	"embed"
	"fmt"
	"io"
	"path"
	"strings"
	"testing/fstest"

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
		"add-unique-datastore-id": migrate.TxMigrationFunc[pgx.Tx](setUniqueID),
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
		version = versionFromFileName(migration.Name())

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
			migrationFn, ok := GoMigrations[version]

			if !ok {
				panic("no go migration registered with name: " + version)
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

func Export(outDir string) (error, fstest.MapFS) {
	migrationFiles, err := migrationFS.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err), nil
	}
	outFiles := make(fstest.MapFS, len(migrationFiles))
	for _, migration := range migrationFiles {
		fileName := path.Join(dir, migration.Name())
		file, err := migrationFS.Open(fileName)
		if err != nil {
			return fmt.Errorf("failed to load migrations: %w", err), nil
		}
		outFiles[path.Join(outDir, migration.Name())] = 
	}
}

func versionFromFileName(fileName string) (version string) {
	version = fileName
	version = version[11:] // trim epoch
	suffixes := []string{".sql", ".sh"}
	for _, s := range suffixes {
		version = strings.TrimSuffix(version, s)
	}
	prefixes := []string{"DDL_", "DML_", "go_", "expand_", "contract_"}
	for _, p := range prefixes {
		version = strings.TrimPrefix(version, p)
	}
	return
}

func initializeVersionTable(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, createVersionTable)
	return err
}
