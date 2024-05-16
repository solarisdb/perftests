package commands

import (
	"fmt"
	"github.com/solarisdb/perftests/pkg/model"
	"github.com/solarisdb/perftests/pkg/server"
	"github.com/solarisdb/perftests/pkg/utils"
	"io"
	"os"
	"strconv"

	"github.com/solarisdb/perftests/pkg/server/configs"
	"github.com/spf13/cobra"
)

var generateCfgCmd = &cobra.Command{
	Use:   "generateCfg [filename | - | auto, op type[sleep|append|cleanup], op params] ",
	Short: "Creates the config: perftests generateCfg perftests.yaml append",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(c *cobra.Command, args []string) error {
		configOutFile := args[0]
		opType := args[1]
		var cfg *model.Config
		var autoFileName string
		switch opType {
		case "sleep":
			autoFileName = "test-scripts/sleep.yaml"
			cfg = server.BuildConfig(server.Sleep, nil)
		case "append":
			concLogs, _ := strconv.Atoi(args[2])
			logSize, _ := strconv.Atoi(args[3])
			writers, _ := strconv.Atoi(args[4])
			batch, _ := strconv.Atoi(args[5])
			msg, _ := strconv.Atoi(args[6])
			cfg = server.BuildConfig(server.Append, &server.AppendCfg{
				ConcurrentLogs:   concLogs,
				LogSize:          logSize,
				WritersForOneLog: writers,
				BatchSize:        batch,
				MsgSize:          msg,
			})
			autoFileName = fmt.Sprintf("test-scripts/append_%s_logs_by_%s_size_%s_writers_batch_%s_by_%s.yaml",
				utils.HumanReadableSizePrecision(float64(concLogs), 0),
				utils.HumanReadableBytesPrecision(float64(logSize), 0),
				utils.HumanReadableSizePrecision(float64(writers), 0),
				utils.HumanReadableSizePrecision(float64(batch), 0),
				utils.HumanReadableBytesPrecision(float64(msg), 0))
		case "cleanup":
			autoFileName = "test-scripts/cleanup.yaml"
			cfg = server.BuildConfig(server.Cleanup, nil)
		}

		jsCfg, err := configs.ToJson(cfg)
		if err != nil {
			return err
		}
		yamlCfg, err := configs.JsonToYaml(string(jsCfg))
		if err != nil {
			return err
		}

		var f io.WriteCloser
		if configOutFile == "-" {
			fmt.Println("Config:")
			f = os.Stdout
		} else if configOutFile == "auto" {
			fmt.Println("write the default config to", autoFileName)
			f, err = os.Create(autoFileName)
			if err != nil {
				return err
			}
			defer f.Close()
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
