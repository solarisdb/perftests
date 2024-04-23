package server

import (
	"context"
	"fmt"
	"strings"

	"github.com/logrange/linker"
	"github.com/solarisdb/perftests/pkg/model"
	"github.com/solarisdb/perftests/pkg/runner"
	"github.com/solarisdb/perftests/pkg/runner/cluster"
	"github.com/solarisdb/perftests/pkg/runner/solaris"
	"github.com/solarisdb/perftests/pkg/version"

	"github.com/solarisdb/solaris/golibs/logging"
)

// Run is an entry point of the server
func Run(ctx context.Context, cfg *model.Config) error {
	_ = setLogLevelByName(cfg.Log.Level)
	logger := logging.NewLogger("runner")
	logger.Infof("Test started (%s)", version.BuildVersionString())
	defer logger.Infof("Test ended")

	testsRunner := runner.NewTestRunner()
	inj := linker.New()
	inj.Register(
		linker.Component{Value: cfg},
		linker.Component{Value: logger},
		linker.Component{Value: runner.NewRegistry()},
		linker.Component{Value: runner.NewSequenceExecutor()},
		linker.Component{Value: runner.NewRepeatExecutor()},
		linker.Component{Value: runner.NewParallelExecutor()},
		linker.Component{Value: runner.NewPauseExecutor()},
		linker.Component{Value: runner.NewAwaitExecutor()},
		linker.Component{Value: runner.NewErrorExecutor()},

		linker.Component{Value: testsRunner},

		//solaris
		linker.Component{Value: solaris.NewMetricsExecutor()},
		linker.Component{Value: solaris.NewConnectExecutor()},
		linker.Component{Value: solaris.NewAppendMsgExecutor()},
		linker.Component{Value: solaris.NewCreateLogExecutor()},
		linker.Component{Value: solaris.NewDeleteLogExecutor()},
		linker.Component{Value: solaris.NewQueryMsgsExecutor()},

		//cluster
		linker.Component{Value: cluster.NewConnectExecutor()},
		linker.Component{Value: cluster.NewFinishExecutor()},
		linker.Component{Value: cluster.NewDeleteClusterExecutor()},
	)
	inj.Init(ctx)
	<-testsRunner.Run(ctx)
	logger.Infof("Stopping ...")
	inj.Shutdown()

	return nil
}

func setLogLevelByName(level string) error {
	l, ok := levelsName[strings.ToUpper(level)]
	if !ok {
		return fmt.Errorf("could not set log level %q, unknown log level name", level)
	}
	logging.SetLevel(l)
	return nil
}

var levelsName = map[string]logging.Level{
	"TRACE": logging.TRACE,
	"DEBUG": logging.DEBUG,
	"INFO":  logging.INFO,
	"WARN":  logging.WARN,
	"ERROR": logging.ERROR,
}
