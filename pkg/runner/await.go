package runner

import (
	"context"
	"fmt"

	"github.com/solarisdb/perftests/pkg/model"
	"github.com/solarisdb/solaris/golibs/errors"
	"github.com/solarisdb/solaris/golibs/logging"
)

type (
	awaitRunner struct {
		exec *awaitExecutor
		name string
	}
	awaitExecutor struct {
		name     string
		Registry *Registry      `inject:""`
		Logger   logging.Logger `inject:""`
	}

	AwaitCfg struct {
		TriggerName string `yaml:"triggerName" json:"triggerName"`
	}
)

const AwaitRunName = "await"

func NewAwaitRunner(exec *awaitExecutor, prefix string) ScenarioRunner {
	return &awaitRunner{exec: exec, name: fmt.Sprintf("%s/%s-%d", prefix, exec.Name(), GetRunnerIndex())}
}

func NewAwaitExecutor() ScenarioExecutor {
	return &awaitExecutor{name: AwaitRunName}
}

func (r *awaitExecutor) Init(ctx context.Context) error {
	return r.Registry.Register(r)
}

func (r *awaitExecutor) Name() string {
	return r.name
}

func (r *awaitExecutor) New(prefix string) ScenarioRunner {
	return NewAwaitRunner(r, prefix)
}

func (r *awaitRunner) RunScenario(ctx context.Context, config *model.ScenarioConfig) <-chan ScenarioResult {
	r.exec.Logger.Debugf("Running scenario %s", r.name)
	defer r.exec.Logger.Debugf("Scenario finished %s", r.name)

	return r.run(ctx, config)
}

func (r *awaitRunner) run(ctx context.Context, config *model.ScenarioConfig) (doneCh chan ScenarioResult) {
	doneCh = make(chan ScenarioResult, 1)
	defer close(doneCh)

	if ctx.Err() != nil {
		doneCh <- &staticScenarioResult{ctx, fmt.Errorf("run context is closed %w", errors.ErrClosed)}
		return
	}

	cfg, err := model.FromScenarioConfig[AwaitCfg](config)
	if err != nil {
		doneCh <- &staticScenarioResult{ctx, fmt.Errorf("failed to parse scenario config %w", err)}
		return
	}

	awaitCtx, _ := ctx.Value(cfg.TriggerName).(context.Context)
	r.exec.Logger.Tracef("Start await %s", cfg.TriggerName)
	<-awaitCtx.Done()
	r.exec.Logger.Tracef("Complete await %s", cfg.TriggerName)

	doneCh <- &staticScenarioResult{ctx: ctx, error: nil}
	return
}
