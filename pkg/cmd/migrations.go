package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/jzelinskie/cobrautil"
	"github.com/spf13/cobra"

	"github.com/authzed/spicedb/internal/datastore/crdb"
	"github.com/authzed/spicedb/internal/datastore/mysql"
	"github.com/authzed/spicedb/internal/datastore/postgres"
	"github.com/authzed/spicedb/internal/datastore/postgres/migrations"
	"github.com/authzed/spicedb/internal/datastore/spanner"
	"github.com/authzed/spicedb/pkg/cmd/server"
	"github.com/authzed/spicedb/pkg/datastore"
	"github.com/authzed/spicedb/pkg/migrate"
)

func RegisterMigrationFlags(cmd *cobra.Command) {
	cmd.Flags().String("datastore-engine", "memory", fmt.Sprintf(`type of datastore to initialize (%s)`, datastore.EngineOptions()))
	cmd.Flags().String("datastore-mysql-table-prefix", "", "prefix to add to the name of all mysql database tables")
	cmd.Flags().Bool("include-versions", true, "if false, omits version tracking from migrations")
}

func NewMigrationCommand(programName string) *cobra.Command {
	return &cobra.Command{
		Use:     "migrations [optional revision]",
		Short:   "Inspect, compute, and export datastore migrations.",
		Long:    fmt.Sprintf("Computes and exports datastore migrations for auditing or integration with external tools. Most users should use \"spicedb migrate\" instead.\nThe special value %q can be used to migrate to the latest revision (also the default).", color.YellowString(migrate.Head)),
		PreRunE: server.DefaultPreRunE(programName),
		RunE:    migrationRun,
		Args:    cobra.MaximumNArgs(1),
	}
}

func migrationRun(cmd *cobra.Command, args []string) error {
	datastoreEngine := cobrautil.MustGetStringExpanded(cmd, "datastore-engine")

	revision := migrate.Head
	if len(args) == 1 {
		revision = args[0]
	}
	switch datastoreEngine {
	case crdb.Engine:
	case postgres.Engine:
		migs := migrations.DatabaseMigrations
		for _, m := range migs
	case mysql.Engine:
	case spanner.Engine:
	default:
		return fmt.Errorf("unknown datastore engine type: %s", datastoreEngine)
	}
	return nil
	// if datastoreEngine == "cockroachdb" {
	// 	// log.Info().Msg("computing cockroachdb datastore")
	// 	//
	// 	// var err error
	// 	// migrationDriver, err := crdbmigrations.NewCRDBDriver(dbURL)
	// 	// if err != nil {
	// 	// 	log.Fatal().Err(err).Msg("unable to create migration driver")
	// 	// }
	// 	// runMigration(cmd.Context(), migrationDriver, crdbmigrations.CRDBMigrations, args[0], timeout)
	// } else if datastoreEngine == "postgres" {
	// 	// log.Info().Msg("migrating postgres datastore")
	// 	//
	// 	// var err error
	// 	// migrationDriver, err := migrations.NewAlembicPostgresDriver(dbURL)
	// 	// if err != nil {
	// 	// 	log.Fatal().Err(err).Msg("unable to create migration driver")
	// 	// }
	// 	// runMigration(cmd.Context(), migrationDriver, migrations.DatabaseMigrations, args[0], timeout)
	// } else if datastoreEngine == "spanner" {
	// 	log.Info().Msg("migrating spanner datastore")
	//
	// 	credFile := cobrautil.MustGetStringExpanded(cmd, "datastore-spanner-credentials")
	// 	var err error
	// 	emulatorHost, err := cmd.Flags().GetString("datastore-spanner-emulator-host")
	// 	if err != nil {
	// 		log.Fatal().Err(err).Msg("unable to get spanner emulator host")
	// 	}
	// 	migrationDriver, err := spannermigrations.NewSpannerDriver(dbURL, credFile, emulatorHost)
	// 	if err != nil {
	// 		log.Fatal().Err(err).Msg("unable to create migration driver")
	// 	}
	// 	runMigration(cmd.Context(), migrationDriver, spannermigrations.SpannerMigrations, args[0], timeout)
	// } else if datastoreEngine == "mysql" {
	// 	log.Info().Msg("migrating mysql datastore")
	//
	// 	var err error
	// 	tablePrefix, err := cmd.Flags().GetString("datastore-mysql-table-prefix")
	// 	if err != nil {
	// 		log.Fatal().Msg(fmt.Sprintf("unable to get table prefix: %s", err))
	// 	}
	//
	// 	migrationDriver, err := mysqlmigrations.NewMySQLDriverFromDSN(dbURL, tablePrefix)
	// 	if err != nil {
	// 		log.Fatal().Err(err).Msg("unable to create migration driver")
	// 	}
	// 	runMigration(cmd.Context(), migrationDriver, mysqlmigrations.DatabaseMigrations, args[0], timeout)
	// } else {
	// 	return fmt.Errorf("unknown datastore engine type: %s", datastoreEngine)
	// }
}
