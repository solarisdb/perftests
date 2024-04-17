package runner

import (
	"context"
	"fmt"
	"time"

	"github.com/solarisdb/perftests/pkg/model"
	"github.com/solarisdb/solaris/golibs/errors"
	"github.com/solarisdb/solaris/golibs/logging"
)

type (
	pauseRunner struct {
		exec *pauseExecutor
		name string
	}
	pauseExecutor struct {
		name     string
		Registry *Registry      `inject:""`
		Logger   logging.Logger `inject:""`
	}

	PauseCfg struct {
		Value string `yaml:"value" json:"value"`
	}
)

const PauseRunName = "pause"

func NewPauseRunner(exec *pauseExecutor, prefix string) ScenarioRunner {
	return &pauseRunner{exec: exec, name: fmt.Sprintf("%s/%s-%d", prefix, exec.Name(), GetRunnerIndex())}
}

func NewPauseExecutor() ScenarioExecutor {
	return &pauseExecutor{name: PauseRunName}
}

func (r *pauseExecutor) Init(ctx context.Context) error {
	return r.Registry.Register(r)
}

func (r *pauseExecutor) Name() string {
	return r.name
}

func (r *pauseExecutor) New(prefix string) ScenarioRunner {
	return NewPauseRunner(r, prefix)
}

func (r *pauseRunner) RunScenario(ctx context.Context, config *model.ScenarioConfig) <-chan ScenarioResult {
	return r.run(ctx, config)
}

func (r *pauseRunner) run(ctx context.Context, config *model.ScenarioConfig) (doneCh chan ScenarioResult) {
	runnerIndex := GetRunnerIndex()
	r.exec.Logger.Debugf("Running scenario %s-%d", r.exec.Name(), runnerIndex)
	defer r.exec.Logger.Debugf("Scenario finished %s-%d", r.exec.Name(), runnerIndex)

	doneCh = make(chan ScenarioResult, 1)
	defer close(doneCh)

	if ctx.Err() != nil {
		doneCh <- &staticScenarioResult{ctx, fmt.Errorf("run context is closed %w", errors.ErrClosed)}
		return
	}

	cfg, err := model.FromScenarioConfig[PauseCfg](config)
	if err != nil {
		doneCh <- &staticScenarioResult{ctx, fmt.Errorf("failed to parse scenario config %w", err)}
		return
	}
	if len(cfg.Value) > 0 {
		pVal, err := time.ParseDuration(cfg.Value)
		if err != nil {
			doneCh <- &staticScenarioResult{ctx, fmt.Errorf("failed to parse pause value %w", err)}
			return
		}
		time.Sleep(pVal)
	}
	doneCh <- &pauseScenarioResult{ctx: ctx, error: nil}
	return
}

type pauseScenarioResult struct {
	ctx   context.Context
	error error
}

var pauseRunnersCounter = "pauseRunnersCounter"

func newPauseScenarioResult(ctx context.Context, err error) ScenarioResult {
	return &pauseScenarioResult{ctx: ctx, error: err}
}

func (r *pauseScenarioResult) Ctx(ctx context.Context) context.Context {
	counter := ctx.Value(pauseRunnersCounter)
	if counter == nil {
		counter = 0
	}
	counter = counter.(int) + 1
	ctx = context.WithValue(ctx, pauseRunnersCounter, counter)
	fmt.Println("print counter: ", counter)
	return ctx
}

func (r *pauseScenarioResult) Error() error {
	return r.error
}
