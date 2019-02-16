package commands

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/thingful/kuzu/pkg/logger"
	"github.com/thingful/kuzu/pkg/postgres"
)

func init() {
	rootCmd.AddCommand(appsCmd)

	appsCmd.Flags().StringP("database-url", "d", "", "Connection string for a PostgreSQL instance")
	appsCmd.Flags().StringP("name", "n", "", "The name of the client application")
	appsCmd.Flags().StringSlice("scope", []string{"timeseries"}, "A comma separated list of scopes, one of: create-users, metadata, or timeseries")

	viper.BindPFlag("database-url", appsCmd.Flags().Lookup("database-url"))
	viper.BindPFlag("name", appsCmd.Flags().Lookup("name"))
	viper.BindPFlag("scope", appsCmd.Flags().Lookup("scope"))
}

var appsCmd = &cobra.Command{
	Use:   "api-key",
	Short: "Create new api keys for client applications",
	Long: `This command allows new api keys to be created for client applications. The
available scopes are: create-users, metadata or timeseries.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		databaseURL := viper.GetString("database-url")
		if databaseURL == "" {
			return errors.New("Must provide a database url")
		}

		verbose := viper.GetBool("verbose")

		name := viper.GetString("name")
		if name == "" {
			return errors.New("Must provide a name for the client app")
		}

		scope := viper.GetStringSlice("scope")
		if len(scope) <= 0 {
			return errors.New("Must provide at least one scope assertion")
		}

		log := logger.NewLogger()

		db := postgres.NewDB(databaseURL, verbose)

		err := db.Start()
		if err != nil {
			return errors.Wrap(err, "failed to start db")
		}

		ctx := logger.ToContext(context.Background(), log)

		app, err := db.CreateApp(ctx, name, scope)
		if err != nil {
			return errors.Wrap(err, "failed to create app")
		}

		fmt.Printf("App created: key: %s\n", app.Key)

		return nil
	},
}
