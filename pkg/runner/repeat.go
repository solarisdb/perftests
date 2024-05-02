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
	repeatRunner struct {
		exec *repeatExecutor
		name string
	}

	repeatExecutor struct {
		name     string
		Registry *Registry      `inject:""`
		Logger   logging.Logger `inject:""`
	}

	RepeatCfg struct {
		Period     string         `yaml:"period,omitempty" json:"period,omitempty"`
		Count      int            `yaml:"count,omitempty" json:"count,omitempty"`
		Action     model.Scenario `yaml:"action" json:"action"`
		Executor   string         `yaml:"executor" json:"executor"`
		SkipErrors bool           `yaml:"skipErrors,omitempty" json:"skipErrors,omitempty"`
	}
)

const (
	RepeatRunName = "repeat"
)

func NewRepeatRunner(exec *repeatExecutor, prefix string) ScenarioRunner {
	return &repeatRunner{exec: exec, name: fmt.Sprintf("%s/%s-%d", prefix, exec.Name(), GetRunnerIndex())}
}

func NewRepeatExecutor() ScenarioExecutor {
	return &repeatExecutor{name: RepeatRunName}
}

func (r *repeatExecutor) Init(ctx context.Context) error {
	return r.Registry.Register(r)
}

func (r *repeatExecutor) Name() string {
	return r.name
}

func (r *repeatExecutor) New(prefix string) ScenarioRunner {
	return NewRepeatRunner(r, prefix)
}

func (r *repeatRunner) RunScenario(ctx context.Context, config *model.ScenarioConfig) <-chan ScenarioResult {
	r.exec.Logger.Debugf("Running scenario %s", r.name)
	defer r.exec.Logger.Debugf("Scenario finished %s", r.name)

	return r.run(ctx, config)
}

func (r *repeatRunner) run(ctx context.Context, config *model.ScenarioConfig) (doneCh chan ScenarioResult) {
	doneCh = make(chan ScenarioResult, 1)
	defer close(doneCh)

	if ctx.Err() != nil {
		doneCh <- &staticScenarioResult{ctx, fmt.Errorf("run context is closed %w", errors.ErrClosed)}
		return
	}

	cfg, err := model.FromScenarioConfig[RepeatCfg](config)
	if err != nil {
		doneCh <- &staticScenarioResult{ctx, fmt.Errorf("failed to parse scenario config %w", err)}
		return
	}
	period := time.Duration(0)
	if len(cfg.Period) > 0 {
		period, err = time.ParseDuration(cfg.Period)
		if err != nil {
			doneCh <- &staticScenarioResult{ctx, fmt.Errorf("failed to parse period %w", err)}
			return
		}
	}
	executor, ok := r.exec.Registry.Get(SequenceRunName)
	if !ok {
		doneCh <- &staticScenarioResult{ctx, fmt.Errorf("failed to get executor %s: %w", SequenceRunName, errors.ErrNotExist)}
		return
	}
	if len(cfg.Executor) > 0 {
		executor, ok = r.exec.Registry.Get(cfg.Executor)
		if !ok {
			doneCh <- &staticScenarioResult{ctx, fmt.Errorf("failed to get executor %s: %w", cfg.Executor, errors.ErrNotExist)}
			return
		}
	}

	var scenarioResult ScenarioResult
	switch executor.Name() {
	case SequenceRunName:
		stepsCnt := cfg.Count
		if period > 0 {
			stepsCnt *= 2
		}
		steps := make([]model.Scenario, stepsCnt)
		for i := 0; i < stepsCnt; i++ {
			steps[i] = cfg.Action
			if len(cfg.Period) > 0 {
				i++
				steps[i] = model.Scenario{
					Name:   PauseRunName,
					Config: model.ToScenarioConfig(&PauseCfg{Value: cfg.Period}),
				}
			}
		}
		secCfg := model.ToScenarioConfig(&SequenceCfg{
			SkipErrors: cfg.SkipErrors,
			Steps:      steps,
		})
		if scenarioResult = <-executor.New(r.name).RunScenario(ctx, secCfg); scenarioResult.Error() != nil {
			doneCh <- scenarioResult
			return
		}
	case ParallelRunName:
		steps := make([]model.Scenario, cfg.Count)
		for i := 0; i < cfg.Count; i++ {
			steps[i] = cfg.Action
		}
		secCfg := model.ToScenarioConfig(&ParallelCfg{
			SkipErrors: cfg.SkipErrors,
			Steps:      steps,
		})
		if scenarioResult = <-executor.New(r.name).RunScenario(ctx, secCfg); scenarioResult.Error() != nil {
			doneCh <- scenarioResult
			return
		}
		if period > 0 {
			time.Sleep(period)
		}
	default:
		doneCh <- &staticScenarioResult{ctx, fmt.Errorf("unsupported executor name %s: %w", cfg.Executor, errors.ErrNotExist)}
		return
	}
	doneCh <- scenarioResult
	return
}
