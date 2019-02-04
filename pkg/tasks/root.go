package tasks

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/thingful/kuzu/pkg/version"
)

var rootCmd = &cobra.Command{
	Use:   version.BinaryName,
	Short: "A tool to do something",
	Long: `An app to do something built by Thingful.

This description spans multiple lines.`,
	Version: version.VersionString(),
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
