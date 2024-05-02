package runner

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/solarisdb/perftests/pkg/model"
	"github.com/solarisdb/solaris/golibs/errors"
	"github.com/solarisdb/solaris/golibs/logging"
)

type (
	// weighted runner executes one (random) step on each execution using weights for each step
	weightedRunner struct {
		exec *weightedExecutor
		name string
	}
	weightedExecutor struct {
		name     string
		Registry *Registry      `inject:""`
		Logger   logging.Logger `inject:""`
	}

	WightedCfg struct {
		Steps   []model.Scenario `yaml:"steps" json:"steps"`
		Weights []uint           `yaml:"weights" json:"weights"`
	}
)

const WeightedRunName = "weighted"

func NewWeightedRunner(exec *weightedExecutor, prefix string) ScenarioRunner {
	return &weightedRunner{exec: exec, name: fmt.Sprintf("%s/%s-%d", prefix, exec.Name(), GetRunnerIndex())}
}

func NewWeightedExecutor() ScenarioExecutor {
	return &weightedExecutor{name: WeightedRunName}
}

func (r *weightedExecutor) Init(ctx context.Context) error {
	return r.Registry.Register(r)
}

func (r *weightedExecutor) Name() string {
	return r.name
}

func (r *weightedExecutor) New(prefix string) ScenarioRunner {
	return NewWeightedRunner(r, prefix)
}

func (r *weightedRunner) RunScenario(ctx context.Context, config *model.ScenarioConfig) <-chan ScenarioResult {
	r.exec.Logger.Debugf("Running scenario %s", r.name)
	defer r.exec.Logger.Debugf("Scenario finished %s", r.name)

	return r.run(ctx, config)
}

func (r *weightedRunner) run(ctx context.Context, config *model.ScenarioConfig) (doneCh chan ScenarioResult) {
	doneCh = make(chan ScenarioResult, 1)
	defer close(doneCh)
	if ctx.Err() != nil {
		doneCh <- &staticScenarioResult{ctx, fmt.Errorf("run context is closed %w", errors.ErrClosed)}
		return
	}
	cfg, err := model.FromScenarioConfig[WightedCfg](config)
	if err != nil {
		doneCh <- &staticScenarioResult{ctx, fmt.Errorf("failed to parse scenario config %w", err)}
		return
	}

	pStep, err := r.chooseStep(cfg.Steps, cfg.Weights)
	if err != nil {
		doneCh <- &staticScenarioResult{ctx, fmt.Errorf("failed select step: %w", err)}
		return
	}

	stepRunner, ok := r.exec.Registry.Get(pStep.Name)
	if !ok {
		doneCh <- &staticScenarioResult{ctx, fmt.Errorf("failed to get step runner: %w", err)}
		return
	}

	// Pass scenario result to parent
	doneCh <- <-stepRunner.New(r.name).RunScenario(ctx, pStep.Config)
	return doneCh
}

func (r *weightedRunner) chooseStep(steps []model.Scenario, weights []uint) (model.Scenario, error) {
	if len(steps) == 0 {
		return model.Scenario{}, fmt.Errorf("must have at least one step defined: %w", errors.ErrNotExist)
	}

	weightsCount := uint(0)
	finalWeights := make([]uint, len(steps))
	shortestLenght := min(len(finalWeights), len(weights))
	// calc weights
	for idx := 0; idx < shortestLenght; idx++ {
		weightsCount += weights[idx]
		finalWeights[idx] = weightsCount
	}
	// fill all non-weighted as 1
	for idx := shortestLenght; idx < len(finalWeights); idx++ {
		weightsCount += 1
		finalWeights[idx] = weightsCount
	}

	// choose one of values
	choose := uint(rand.Int31n(int32(weightsCount + 1)))
	for idx := 0; idx < len(finalWeights); idx++ {
		if choose <= finalWeights[idx] {
			return steps[idx], nil
		}
	}

	return model.Scenario{}, errors.ErrInternal
}
