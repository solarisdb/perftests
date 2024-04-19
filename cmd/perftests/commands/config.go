package commands

import (
	"fmt"
	"io"
	"os"

	"github.com/solarisdb/perftests/pkg/server"
	"github.com/solarisdb/perftests/pkg/server/configs"
	"github.com/spf13/cobra"
)

var generateCfgCmd = &cobra.Command{
	Use:   "generateCfg [filename | -] ",
	Short: "Creates the config: perftests generateCfg perftests.yaml",
	Args:  cobra.ExactArgs(1),
	RunE: func(c *cobra.Command, args []string) error {
		cfg := server.GetDefaultConfig()

		jsCfg, err := configs.ToJson(cfg)
		if err != nil {
			return err
		}
		yamlCfg, err := configs.JsonToYaml(string(jsCfg))
		if err != nil {
			return err
		}

		var f io.WriteCloser
		configOutFile := args[0]
		if configOutFile == "-" {
			fmt.Println("Config:")
			f = os.Stdout
		} else {
			fmt.Println("write the default config to", configOutFile)
			f, err = os.Create(configOutFile)
			if err != nil {
				return err
			}
			defer f.Close()
		}
		_, err = f.Write(yamlCfg)
		if err != nil {
			return err
		}
		return nil
	},
}
