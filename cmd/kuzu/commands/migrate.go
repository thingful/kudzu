package commands

import (
	"errors"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/thingful/kuzu/pkg/logger"
	"github.com/thingful/kuzu/pkg/postgres"
)

func init() {
	rootCmd.AddCommand(migrateCmd)
	migrateCmd.AddCommand(migrateNewCmd)
	migrateCmd.AddCommand(migrateUpCmd)
	migrateCmd.AddCommand(migrateDownCmd)

	migrateCmd.PersistentFlags().StringP("database-url", "d", "", "Connection string for a PostgreSQL instance")
	viper.BindPFlag("database-url", migrateCmd.PersistentFlags().Lookup("database-url"))

	migrateNewCmd.Flags().String("dir", "pkg/migrations/sql", "The directory into which migration files must be written")

	migrateDownCmd.Flags().Int("steps", 1, "Number of down migration steps to run")
	migrateDownCmd.Flags().Bool("all", false, "Whether to run all down migrations")

	viper.BindPFlag("steps", migrateDownCmd.Flags().Lookup("steps"))
	viper.BindPFlag("all", migrateDownCmd.Flags().Lookup("all"))
}

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Manage Postgres migrations",
	Long: `This command provides tools for working with migrations for Postgres.

Up migrations are run automatically when the application boots, but here we
also include commands for creating properly named migration files, and
commands for running down migrations.`,
}

var migrateNewCmd = &cobra.Command{
	Use:   "new",
	Short: "Create a new Postgres migration",
	Long: `This command creates a new pair of matching migration files in the
correct directory that are properly named. The desired name of the migration
should be passed via a positional argument after the new command.

For example:

		$ kuzu migrate new AddUserTable`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := cmd.Flags().GetString("dir")
		if err != nil {
			return err
		}

		log := logger.NewLogger()

		return postgres.NewMigration(dir, args[0], log)
	},
}

var migrateUpCmd = &cobra.Command{
	Use:   "up",
	Short: "Run all up migrations",
	Long:  `This command attempts to run all up migrations against Postgres.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		databaseURL := viper.GetString("database-url")
		if databaseURL == "" {
			return errors.New("Must provide a database url")
		}

		log := logger.NewLogger()

		db, err := postgres.Open(databaseURL)
		if err != nil {
			return err
		}

		return postgres.MigrateUp(db.DB, log)
	},
}

var migrateDownCmd = &cobra.Command{
	Use:   "down",
	Short: "Run down migrations",
	Long:  `This command attempts to run down migrations against Postgres.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		databaseURL := viper.GetString("database-url")
		if databaseURL == "" {
			return errors.New("Must provide a database url")
		}

		steps := viper.GetInt("steps")
		all := viper.GetBool("all")

		if steps == 0 && all == false {
			return errors.New("Must supply either a number of steps or the all flag")
		}

		log := logger.NewLogger()

		db, err := postgres.Open(databaseURL)
		if err != nil {
			return err
		}

		if all {
			return postgres.MigrateDownAll(db.DB, log)
		}

		return postgres.MigrateDown(db.DB, steps, log)
	},
}
