package migrations

import (
	"bytes"
	"context"
	"database/sql"
	"embed"
	"io"
	"path"
	"strings"

	"github.com/authzed/spicedb/pkg/migrate"
)

// Wrapper makes it possible to forward the table schema needed for MySQL MigrationFunc to run
type Wrapper struct {
	db     *sql.DB
	tables *tables
}

// TxWrapper makes it possible to forward the table schema to a transactional migration func.
type TxWrapper struct {
	tx     *sql.Tx
	tables *tables
}

const (
	dir = "migrations"

	createVersionTable = `CREATE TABLE mysql_migration_version (
    id int(11) NOT NULL PRIMARY KEY,
    _meta_version_ VARCHAR(255) NOT NULL) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`
)

var (
	//go:embed migrations/*
	migrationFS embed.FS

	// DatabaseMigrations implements a migration manager for the Postgres Driver.
	DatabaseMigrations = migrate.NewManager[*MySQLDriver, Wrapper, TxWrapper]()

	GoMigrations = map[string]any{}
)

func init() {
	version := "init"
	if err := DatabaseMigrations.Register("0000000000_DDL_expand_initialize_versions.sql", version, "", nil, initializeVersionTable); err != nil {
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
		version = strings.TrimSuffix(version, ".sh")
		version = version[11:] // trim epoch
		version = strings.TrimPrefix(version, "DDL_")
		version = strings.TrimPrefix(version, "DML_")
		version = strings.TrimPrefix(version, "go_")
		version = strings.TrimPrefix(version, "expand_")
		version = strings.TrimPrefix(version, "contract_")

		if strings.HasSuffix(migration.Name(), "sql") {
			sqlBytes, err := io.ReadAll(file)
			if err != nil {
				panic("failed to load migration " + fileName + ": " + err.Error())
			}
			if err := DatabaseMigrations.Register(fileName, version, previous, func(ctx context.Context, conn Wrapper) error {
				for _, stmt := range bytes.Split(sqlBytes, []byte(";")) {
					stmt := bytes.TrimSpace(stmt)
					if len(stmt) == 0 {
						continue
					}
					_, err := conn.db.ExecContext(ctx, string(stmt))
					if err != nil {
						return err
					}
				}
				return nil
			}, nil); err != nil {
				panic("failed to register migration " + fileName + ": " + err.Error())
			}
		} else if strings.HasSuffix(migration.Name(), "sh") {
			migrationFn, ok := GoMigrations[version]

			if !ok {
				panic("no go migration registered with name: " + version)
			}

			switch f := migrationFn.(type) {
			case migrate.TxMigrationFunc[TxWrapper]:
				if err := DatabaseMigrations.Register(fileName, version, previous, nil, f); err != nil {
					panic("failed to register migration " + fileName + ": " + err.Error())
				}
			case migrate.MigrationFunc[Wrapper]:
				if err := DatabaseMigrations.Register(fileName, version, previous, f, nil); err != nil {
					panic("failed to register migration " + fileName + ": " + err.Error())
				}
			default:
				panic("unknown migration type")
			}
		}
	}
}

func initializeVersionTable(ctx context.Context, tx TxWrapper) error {
	_, err := tx.tx.ExecContext(ctx, createVersionTable)
	return err
}
