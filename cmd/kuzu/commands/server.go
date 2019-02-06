package commands

import (
	"errors"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/thingful/kuzu/pkg/app"
)

func init() {
	rootCmd.AddCommand(serverCmd)

	serverCmd.Flags().StringP("addr", "a", "0.0.0.0:3001", "Specify the address to which the server binds")
	serverCmd.Flags().StringP("database-url", "d", "", "Connection string for a PostgreSQL instance")
	serverCmd.Flags().Int("client-timeout", 5, "HTTP client timeout in seconds")

	viper.BindPFlag("addr", serverCmd.Flags().Lookup("addr"))
	viper.BindPFlag("database-url", serverCmd.Flags().Lookup("database-url"))
	viper.BindPFlag("client-timeout", serverCmd.Flags().Lookup("client-timeout"))
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the server",
	Long:  `Starts the server running along with a background indexer.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		addr := viper.GetString("addr")
		if addr == "" {
			return errors.New("Must provide a bind address")
		}

		databaseURL := viper.GetString("database-url")
		if databaseURL == "" {
			return errors.New("Must provide a database url")
		}

		clientTimeout := viper.GetInt("client-timeout")
		if clientTimeout == 0 {
			return errors.New("Must provide a non-zero client timeout")
		}

		a := app.NewApp(&app.Config{
			Addr:          addr,
			DatabaseURL:   databaseURL,
			ClientTimeout: clientTimeout,
		})

		return a.Start()
	},
}
