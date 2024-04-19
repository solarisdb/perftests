package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	SilenceUsage: true,
}

func init() {
	rootCmd.InitDefaultHelpCmd()
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(generateCfgCmd)
	rootCmd.AddCommand(versionCmd)
}

// Execute allows to execute cobra commands
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed: %s\n", err)
		os.Exit(1)
	}
}
