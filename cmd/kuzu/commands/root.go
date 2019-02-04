package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/thingful/kuzu/pkg/version"
)

var rootCmd = &cobra.Command{
	Use:     version.BinaryName,
	Short:   "A server and indexer for GROW",
	Long:    `The updated server and indexer component for GROW.`,
	Version: version.VersionString(),
}

// Execute is the main entry point for our cobra commands
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
