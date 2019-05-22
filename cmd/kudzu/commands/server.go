package commands

import (
	"context"
	"errors"
	"time"

	"github.com/lestrrat-go/backoff"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/thingful/kudzu/pkg/app"
	"golang.org/x/time/rate"
)

func init() {
	rootCmd.AddCommand(serverCmd)

	serverCmd.Flags().StringP("addr", "a", "0.0.0.0:3001", "Specify the address to which the server binds")
	serverCmd.Flags().StringP("database-url", "d", "", "Connection string for a PostgreSQL instance")
	serverCmd.Flags().Int("client-timeout", 10, "HTTP client timeout in seconds")
	serverCmd.Flags().Int("delay", 10, "Minimum delay time in seconds for indexer task")
	serverCmd.Flags().String("thingful-url", "https://api.thingful.net", "The server URL at which the Thingful API is available")
	serverCmd.Flags().String("thingful-key", "", "A valid Thingful API key")
	serverCmd.Flags().Int("concurrency", 3, "The number of parallel go routines to spawn when fetching from Thingful")
	serverCmd.Flags().Bool("no-indexer", false, "If present stop the indexer from running")
	serverCmd.Flags().Int("server-timeout", 5, "HTTP server timeout in seconds")
	serverCmd.Flags().Float64("rate", 4, "The base rate permitted per client in req/sec")
	serverCmd.Flags().Int("burst", 8, "The burstable API rate per client in req/sec")
	serverCmd.Flags().Int("expiry", 60, "The interval in seconds with which we sweep the rate limit storage to free resources")

	viper.BindPFlag("addr", serverCmd.Flags().Lookup("addr"))
	viper.BindPFlag("database-url", serverCmd.Flags().Lookup("database-url"))
	viper.BindPFlag("client-timeout", serverCmd.Flags().Lookup("client-timeout"))
	viper.BindPFlag("delay", serverCmd.Flags().Lookup("delay"))
	viper.BindPFlag("thingful-url", serverCmd.Flags().Lookup("thingful-url"))
	viper.BindPFlag("thingful-key", serverCmd.Flags().Lookup("thingful-key"))
	viper.BindPFlag("concurrency", serverCmd.Flags().Lookup("concurrency"))
	viper.BindPFlag("no-indexer", serverCmd.Flags().Lookup("no-indexer"))
	viper.BindPFlag("server-timeout", serverCmd.Flags().Lookup("server-timeout"))
	viper.BindPFlag("rate", serverCmd.Flags().Lookup("rate"))
	viper.BindPFlag("burst", serverCmd.Flags().Lookup("burst"))
	viper.BindPFlag("expiry", serverCmd.Flags().Lookup("expiry"))
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

		serverTimeout := viper.GetInt("server-timeout")
		if serverTimeout == 0 {
			return errors.New("Must provide a non-zero server timeout")
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

		baseRate := viper.GetFloat64("rate")
		if baseRate == 0 {
			return errors.New("Must specify a non-zero rate")
		}

		expiry := viper.GetInt("expiry")
		if expiry == 0 {
			return errors.New("Must specify a non-zero rate limit expiry")
		}

		e := backoff.ExecuteFunc(func(_ context.Context) error {
			a := app.NewApp(&app.Config{
				Addr:          addr,
				DatabaseURL:   databaseURL,
				ClientTimeout: clientTimeout,
				Verbose:       verbose,
				Delay:         delay,
				ThingfulURL:   thingfulURL,
				ThingfulKey:   thingfulKey,
				Concurrency:   viper.GetInt("concurrency"),
				NoIndexer:     viper.GetBool("no-indexer"),
				ServerTimeout: serverTimeout,
				Rate:          rate.Limit(baseRate),
				Burst:         viper.GetInt("burst"),
				Expiry:        time.Duration(expiry) * time.Second,
			})

			return a.Start()
		})

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		policy := backoff.NewExponential()
		return backoff.Retry(ctx, policy, e)
	},
}
