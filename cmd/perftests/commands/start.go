package commands

import (
	"os"
	"strings"
	"syscall"

	"github.com/solarisdb/perftests/pkg/model"
	"github.com/solarisdb/perftests/pkg/server"
	"github.com/solarisdb/perftests/pkg/server/configs"
	"github.com/solarisdb/solaris/golibs/context"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start config.yaml",
	Short: "Starts the service: perftests start {cfg_file_names}...}",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(c *cobra.Command, args []string) error {
		defaultCfg := server.GetDefaultConfig()
		envVarsCfg, err := configs.LoadFromEnvVars()
		if err != nil {
			return err
		}
		appCfg := &model.Config{}
		if err := configs.Merge(appCfg, &envVarsCfg); err != nil {
			return err
		}
		for _, arg := range args {
			configFile := strings.TrimSpace(arg)
			fileCfg, err := configs.LoadFromFile(configFile)
			if err != nil {
				return err
			}
			if err := configs.Merge(appCfg, &fileCfg); err != nil {
				return err
			}
		}
		if err := configs.Merge(appCfg, defaultCfg); err != nil {
			return err
		}
		mainCtx := context.NewSignalsContext(os.Interrupt, syscall.SIGTERM)
		return server.Run(mainCtx, appCfg)
	},
}
