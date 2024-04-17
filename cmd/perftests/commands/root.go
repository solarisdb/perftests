package commands

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

var rootCmd = &cobra.Command{
	SilenceUsage: true,
}

func init() {
	rootCmd.InitDefaultHelpCmd()
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(defaultCfgCmd)
	rootCmd.AddCommand(versionCmd)
}

// Execute allows to execute cobra commands
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed: %s\n", err)
		os.Exit(1)
	}
}
