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
	createLog struct {
		exec *createLogExecutor
		name string
	}

	createLogExecutor struct {
		name     string
		Registry *runner.Registry `inject:""`
		Logger   logging.Logger   `inject:""`
	}

	CreateLogCfg struct {
		Tags map[string]string `yaml:"tags" json:"tags"`
	}

	createLogScenarioResult struct {
		logID string
	}
)

const (
	solarisLog    = "solarisLog"
	CreateLogName = "solaris.createLog"
)

func NewCreateLog(exec *createLogExecutor, prefix string) runner.ScenarioRunner {
	return &createLog{exec: exec, name: fmt.Sprintf("%s/%s-%d", prefix, exec.Name(), runner.GetRunnerIndex())}
}

func NewCreateLogExecutor() runner.ScenarioExecutor {
	return &createLogExecutor{name: CreateLogName}
}

func (r *createLogExecutor) Init(ctx context.Context) error {
	return r.Registry.Register(r)
}

func (r *createLogExecutor) Name() string {
	return r.name
}

func (r *createLogExecutor) New(prefix string) runner.ScenarioRunner {
	return NewCreateLog(r, prefix)
}

func (r *createLog) RunScenario(ctx context.Context, config *model.ScenarioConfig) <-chan runner.ScenarioResult {
	r.exec.Logger.Debugf("Running scenario %s", r.name)
	defer r.exec.Logger.Debugf("Scenario finished %s", r.name)

	return r.run(ctx, config)
}

func (r *createLog) run(ctx context.Context, config *model.ScenarioConfig) (doneCh chan runner.ScenarioResult) {
	doneCh = make(chan runner.ScenarioResult, 1)
	defer close(doneCh)

	if ctx.Err() != nil {
		doneCh <- runner.NewStaticScenarioResult(ctx, fmt.Errorf("run context is closed %w", errors.ErrClosed))
		return
	}

	cfg, err := model.FromScenarioConfig[CreateLogCfg](config)
	if err != nil {
		doneCh <- runner.NewStaticScenarioResult(ctx, fmt.Errorf("failed to parse scenario config %w", err))
		return
	}

	clnt, _ := ctx.Value(solarisClnt).(solaris.ServiceClient)
	if clnt == nil {
		doneCh <- runner.NewStaticScenarioResult(ctx, fmt.Errorf("solaris service not found"))
		return
	}

	log, err := clnt.CreateLog(ctx, &solaris.Log{
		Tags: cfg.Tags,
	})
	if err != nil {
		doneCh <- runner.NewStaticScenarioResult(ctx, fmt.Errorf("failed to create a log"))
		return
	}
	doneCh <- &createLogScenarioResult{
		log.ID,
	}
	return
}

func (r *createLogScenarioResult) Ctx(ctx context.Context) context.Context {
	client := ctx.Value(solarisLog)
	if client == nil {
		ctx = context.WithValue(ctx, solarisLog, r.logID)
	}
	return ctx
}

func (r *createLogScenarioResult) Error() error {
	return nil
}
