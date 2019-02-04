package commands

import (
	"github.com/spf13/cobra"

	"github.com/thingful/kuzu/pkg/logger"
	"github.com/thingful/kuzu/pkg/postgres"
)

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

		logger := logger.NewLogger()

		return postgres.NewMigration(dir, args[0], logger)
	},
}

func init() {
	rootCmd.AddCommand(migrateCmd)
	migrateCmd.AddCommand(migrateNewCmd)

	migrateNewCmd.Flags().String("dir", "pkg/migrations/sql", "The directory into which migration files must be written")
}
