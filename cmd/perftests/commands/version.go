package commands

import (
	"fmt"

	"github.com/solarisdb/perftests/pkg/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use: "version",
	Run: func(c *cobra.Command, args []string) {
		fmt.Println(version.BuildVersionString())
	},
}
