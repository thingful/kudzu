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
	serverCmd.Flags().Int("client-timeout", 10, "HTTP client timeout in seconds")
	serverCmd.Flags().BoolP("verbose", "v", false, "Boolean flag to enable verbose logging")
	serverCmd.Flags().Int("delay", 10, "Minimum delay time in seconds for indexer task")
	serverCmd.Flags().String("thingful-url", "https://api.thingful.net", "The server URL at which the Thingful API is available")
	serverCmd.Flags().String("thingful-key", "", "A valid Thingful API key")

	viper.BindPFlag("addr", serverCmd.Flags().Lookup("addr"))
	viper.BindPFlag("database-url", serverCmd.Flags().Lookup("database-url"))
	viper.BindPFlag("client-timeout", serverCmd.Flags().Lookup("client-timeout"))
	viper.BindPFlag("verbose", serverCmd.Flags().Lookup("verbose"))
	viper.BindPFlag("delay", serverCmd.Flags().Lookup("delay"))
	viper.BindPFlag("thingful-url", serverCmd.Flags().Lookup("thingful-url"))
	viper.BindPFlag("thingful-key", serverCmd.Flags().Lookup("thingful-key"))
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

		verbose := viper.GetBool("verbose")

		delay := viper.GetInt("delay")
		if delay == 0 {
			return errors.New("Must provide a non-zero delay value")
		}

		thingfulURL := viper.GetString("thingful-url")
		if thingfulURL == "" {
			return errors.New("Must specify the Thingful API URL")
		}

		thingfulKey := viper.GetString("thingful-key")
		if thingfulKey == "" {
			return errors.New("Must specify the Thingful API key")
		}

		a := app.NewApp(&app.Config{
			Addr:          addr,
			DatabaseURL:   databaseURL,
			ClientTimeout: clientTimeout,
			Verbose:       verbose,
			Delay:         delay,
			ThingfulURL:   thingfulURL,
			ThingfulKey:   thingfulKey,
		})

		return a.Start()
	},
}
