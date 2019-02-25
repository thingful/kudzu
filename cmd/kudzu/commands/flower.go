package commands

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/thingful/kudzu/pkg/client"
	"github.com/thingful/kudzu/pkg/flowerpower"
)

func init() {
	rootCmd.AddCommand(flowerCmd)
}

var flowerCmd = &cobra.Command{
	Use:   "flower",
	Short: "Interrogate flowerpower for given identities",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := client.NewClient(5, true)
		locations, err := flowerpower.GetLocations(context.Background(), client, args[0])
		if err != nil {
			return err
		}

		fmt.Printf("User has %v sensors\n", len(locations))

		return nil
	},
}
