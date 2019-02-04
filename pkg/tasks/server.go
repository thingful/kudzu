package tasks

import (
	"github.com/spf13/cobra"

	"github.com/thingful/kuzu/pkg/http"
)

func init() {
	rootCmd.AddCommand(serverCmd)

	serverCmd.Flags().StringP("addr", "a", "0.0.0.0:8080", "Specify the address to which the server binds")
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start http server",
	Long: `Starts a simple http server to verify that the process runs
continuously within the container.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		addr, err := cmd.Flags().GetString("addr")
		if err != nil {
			return err
		}

		s := http.NewServer(addr)

		s.Start()

		return nil
	},
}
