package commands

import (
	"log"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/thingful/kudzu/pkg/version"
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
	viper.SetEnvPrefix("kudzu")
	viper.AutomaticEnv()
	replacer := strings.NewReplacer("-", "_")
	viper.SetEnvKeyReplacer(replacer)

	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Boolean flag to enable verbose logging")
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
}

// Execute is the main entry point for our cobra commands
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
