package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/thingful/kuzu/pkg/version"
)

var rootCmd = &cobra.Command{
	Use:   version.BinaryName,
	Short: "A server and indexer for GROW",
	Long: `The updated server and indexer component for GROW.

This component exposes an HTTP API for servicing the collaboration hub
primarily, it also is responsible for indexing Parrot or other resources,
and pushing the collected data into the core Thingful API.`,
	Version: version.VersionString(),
}

func init() {
	viper.SetEnvPrefix("KUZU")
	viper.AutomaticEnv()
	replacer := strings.NewReplacer("-", "_")
	viper.SetEnvKeyReplacer(replacer)
}

// Execute is the main entry point for our cobra commands
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
