package runner

import (
	"context"
	"fmt"

	"github.com/solarisdb/perftests/pkg/model"
	"github.com/solarisdb/solaris/golibs/errors"
	"github.com/solarisdb/solaris/golibs/logging"
)

type (
	errorRunner struct {
		exec *errorExecutor
		name string
	}
	errorExecutor struct {
		name     string
		Registry *Registry      `inject:""`
		Logger   logging.Logger `inject:""`
	}

	ErrorCfg struct {
		Error string `yaml:"error" json:"error"`
	}
)

const ErrorRunName = "error"

func NewErrorRunner(exec *errorExecutor, prefix string) ScenarioRunner {
	return &errorRunner{exec: exec, name: fmt.Sprintf("%s/%s-%d", prefix, exec.Name(), GetRunnerIndex())}
}

func NewErrorExecutor() ScenarioExecutor {
	return &errorExecutor{name: ErrorRunName}
}

func (r *errorExecutor) Init(ctx context.Context) error {
	return r.Registry.Register(r)
}

func (r *errorExecutor) Name() string {
	return r.name
}

func (r *errorExecutor) New(prefix string) ScenarioRunner {
	return NewErrorRunner(r, prefix)
}

func (r *errorRunner) RunScenario(ctx context.Context, config *model.ScenarioConfig) <-chan ScenarioResult {
	r.exec.Logger.Debugf("Running scenario %s", r.name)
	defer r.exec.Logger.Debugf("Scenario finished %s", r.name)

	return r.run(ctx, config)
}

func (r *errorRunner) run(ctx context.Context, config *model.ScenarioConfig) (doneCh chan ScenarioResult) {
	doneCh = make(chan ScenarioResult, 1)
	defer close(doneCh)
	if ctx.Err() != nil {
		doneCh <- &staticScenarioResult{ctx, fmt.Errorf("run context is closed %w", errors.ErrClosed)}
		return
	}
	cfg, err := model.FromScenarioConfig[ErrorCfg](config)
	if err != nil {
		doneCh <- &staticScenarioResult{ctx, fmt.Errorf("failed to parse scenario config %w", err)}
		return
	}
	doneCh <- &staticScenarioResult{ctx: ctx, error: fmt.Errorf(cfg.Error)}
	return
}
