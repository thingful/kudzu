package commands

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/thingful/kuzu/pkg/app"
)

func init() {
	rootCmd.AddCommand(serverCmd)

	serverCmd.Flags().StringP("addr", "a", "0.0.0.0:3001", "Specify the address to which the server binds")
	serverCmd.Flags().StringP("database-url", "d", "", "Connection string for a PostgreSQL instance")
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the server",
	Long:  `Starts the server running along with a background indexer.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		addr, err := cmd.Flags().GetString("addr")
		if err != nil {
			return err
		}

		//connStr, err := cmd.Flags().GetString("database-url")
		//if err != nil {
		//	return err
		//}

		connStr := os.Getenv("DATABASE_URL")

		a := app.NewApp(addr, connStr)

		return a.Start()
	},
}
