package solaris

import (
	"context"
	"fmt"

	"github.com/solarisdb/perftests/pkg/model"
	"github.com/solarisdb/perftests/pkg/runner"
	"github.com/solarisdb/solaris/api/gen/solaris/v1"
	"github.com/solarisdb/solaris/golibs/errors"
	"github.com/solarisdb/solaris/golibs/logging"
)

type (
	deleteLog struct {
		exec *deleteLogExecutor
		name string
	}

	deleteLogExecutor struct {
		name     string
		Registry *runner.Registry `inject:""`
		Logger   logging.Logger   `inject:""`
	}

	deleteLogScenarioResult struct {
		logID string
	}
)

const (
	DeleteLogName = "solaris.deleteLog"
)

func NewDeleteLog(exec *deleteLogExecutor, prefix string) runner.ScenarioRunner {
	return &deleteLog{exec: exec, name: fmt.Sprintf("%s/%s-%d", prefix, exec.Name(), runner.GetRunnerIndex())}
}

func NewDeleteLogExecutor() runner.ScenarioExecutor {
	return &deleteLogExecutor{name: DeleteLogName}
}

func (r *deleteLogExecutor) Init(ctx context.Context) error {
	return r.Registry.Register(r)
}

func (r *deleteLogExecutor) Name() string {
	return r.name
}

func (r *deleteLogExecutor) New(prefix string) runner.ScenarioRunner {
	return NewDeleteLog(r, prefix)
}

func (r *deleteLog) RunScenario(ctx context.Context, config *model.ScenarioConfig) <-chan runner.ScenarioResult {
	r.exec.Logger.Debugf("Running scenario %s", r.name)
	defer r.exec.Logger.Debugf("Scenario finished %s", r.name)

	return r.run(ctx, config)
}

func (r *deleteLog) run(ctx context.Context, _ *model.ScenarioConfig) (doneCh chan runner.ScenarioResult) {
	doneCh = make(chan runner.ScenarioResult, 1)
	defer close(doneCh)

	if ctx.Err() != nil {
		doneCh <- runner.NewStaticScenarioResult(ctx, fmt.Errorf("run context is closed %w", errors.ErrClosed))
		return
	}

	clnt, _ := ctx.Value(solarisClnt).(solaris.ServiceClient)
	if clnt == nil {
		doneCh <- runner.NewStaticScenarioResult(ctx, fmt.Errorf("solaris service not found"))
		return
	}
	log, _ := ctx.Value(solarisLog).(string)
	if len(log) == 0 {
		doneCh <- runner.NewStaticScenarioResult(ctx, fmt.Errorf("solaris log not found"))
		return
	}
	_, err := clnt.DeleteLogs(ctx, &solaris.DeleteLogsRequest{
		Condition: fmt.Sprintf("logID='%s'", log),
	})
	if err != nil {
		doneCh <- runner.NewStaticScenarioResult(ctx, fmt.Errorf("failed to delete log %s: %w", log, err))
		return
	}
	doneCh <- &deleteLogScenarioResult{}
	return
}

func (r *deleteLogScenarioResult) Ctx(ctx context.Context) context.Context {
	return context.WithValue(ctx, solarisLog, nil)
}

func (r *deleteLogScenarioResult) Error() error {
	return nil
}
