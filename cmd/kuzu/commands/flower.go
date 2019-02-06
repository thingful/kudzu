package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/thingful/kuzu/pkg/client"
	"github.com/thingful/kuzu/pkg/flowerpower"
	"github.com/thingful/kuzu/pkg/logger"
)

func init() {
	rootCmd.AddCommand(flowerCmd)
}

var flowerCmd = &cobra.Command{
	Use:   "flower",
	Short: "Interrogate flowerpower for given identities",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := client.NewClient(5, logger.NewLogger())
		count, err := flowerpower.SensorCount(client, args[0])
		if err != nil {
			return err
		}

		fmt.Printf("User has %v sensors\n", count)

		return nil
	},
}
