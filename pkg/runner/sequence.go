package runner

import (
	"context"
	"fmt"
	"time"

	"github.com/solarisdb/perftests/pkg/model"
	"github.com/solarisdb/solaris/golibs/container"
	"github.com/solarisdb/solaris/golibs/errors"
	"github.com/solarisdb/solaris/golibs/logging"
)

type (
	sequenceRunner struct {
		exec *sequenceExecutor
		name string
	}

	sequenceExecutor struct {
		Registry *Registry      `inject:""`
		Logger   logging.Logger `inject:""`
	}

	SequenceCfg struct {
		SkipErrors        bool             `yaml:"skipErrors,omitempty" json:"skipErrors,omitempty"`
		Steps             []model.Scenario `yaml:"steps" json:"steps"`
		TimeoutMetricName string           `yaml:"timeoutMetric,omitempty" json:"timeoutMetric,omitempty"`
		RpsMetricName     string           `yaml:"rpsMetric,omitempty" json:"rpsMetric,omitempty"`
	}

	seqScenarioResult struct {
		results    []ScenarioResult
		runner     *sequenceRunner
		skipErrors bool
	}
)

const SequenceRunName = "sequence"

func NewSequenceRunner(exec *sequenceExecutor, prefix string) ScenarioRunner {
	return &sequenceRunner{exec: exec, name: fmt.Sprintf("%s/%s-%d", prefix, exec.Name(), GetRunnerIndex())}
}

func NewSequenceExecutor() ScenarioExecutor {
	return &sequenceExecutor{}
}

func (r *sequenceExecutor) Init(ctx context.Context) error {
	return r.Registry.Register(r)
}

func (r *sequenceExecutor) Name() string {
	return SequenceRunName
}

func (r *sequenceExecutor) New(prefix string) ScenarioRunner {
	return NewSequenceRunner(r, prefix)
}

func (r *sequenceRunner) RunScenario(ctx context.Context, config *model.ScenarioConfig) <-chan ScenarioResult {
	r.exec.Logger.Debugf("Running scenario %s", r.name)
	defer r.exec.Logger.Debugf("Scenario finished %s", r.name)

	return r.run(ctx, config)
}

func (r *sequenceRunner) run(mctx context.Context, config *model.ScenarioConfig) (doneCh chan ScenarioResult) {
	doneCh = make(chan ScenarioResult, 1)
	defer close(doneCh)

	if mctx.Err() != nil {
		doneCh <- &staticScenarioResult{mctx, fmt.Errorf("run context is closed %w", errors.ErrClosed)}
		return
	}

	cfg, err := model.FromScenarioConfig[SequenceCfg](config)
	if err != nil {
		doneCh <- &staticScenarioResult{mctx, fmt.Errorf("failed to parse scenario config %w", err)}
		return
	}

	stepRes := []ScenarioResult{}
	lastSeqStepRes := newSeqScenarioResult(stepRes, r, cfg.SkipErrors)
	ctx := mctx
	for indx, step := range cfg.Steps {
		stepRunner, ok := r.exec.Registry.Get(step.Name)
		if !ok {
			doneCh <- &staticScenarioResult{mctx, fmt.Errorf("failed to get runner for step \"%s\" index[%d]: %w", step.Name, indx, errors.ErrNotExist)}
			return
		}
		timeOutM, _ := GetDurationMetric(ctx, cfg.TimeoutMetricName)
		rpsM, _ := GetRateMetric(ctx, cfg.RpsMetricName)
		start := time.Now()
		stepResCh := stepRunner.New(r.name).RunScenario(ctx, step.Config)
		stepRes = append(stepRes, <-stepResCh)
		dur := time.Since(start)
		if timeOutM != nil {
			timeOutM.Add(dur.Nanoseconds())
		}
		if rpsM != nil {
			rpsM.Add(1, dur)
		}
		lastSeqStepRes = newSeqScenarioResult(stepRes, r, cfg.SkipErrors)
		ctx = lastSeqStepRes.Ctx(mctx)
		if stepErr := lastSeqStepRes.Error(); stepErr != nil {
			doneCh <- &staticScenarioResult{mctx, fmt.Errorf("failed run of runner \"%s\" index[%d]: %w", step.Name, indx, stepErr)}
			return
		}
	}
	doneCh <- lastSeqStepRes
	return
}

func newSeqScenarioResult(results []ScenarioResult, runner *sequenceRunner, skipErrors bool) *seqScenarioResult {
	return &seqScenarioResult{results: results, runner: runner, skipErrors: skipErrors}
}

func (r *seqScenarioResult) Ctx(ctx context.Context) context.Context {
	for index, stepRes := range r.results {
		if stepErr := stepRes.Error(); stepErr == nil {
			ctx = stepRes.Ctx(ctx)
		} else if r.skipErrors {
			eList := container.CopyMap(ctx.Value(SkippedErrorsMap).(map[string]error))
			eList[fmt.Sprintf("%s/step#%d", r.runner.name, index)] = stepErr
			ctx = context.WithValue(ctx, SkippedErrorsMap, eList)
		}
	}
	return ctx
}

func (r *seqScenarioResult) Error() error {
	var err error
	if r.skipErrors {
		return nil
	}
	for index, stepRes := range r.results {
		if stepRes.Error() != nil {
			if err != nil {
				err = fmt.Errorf("%s,\n{failed sequence step#%d caused by: %s}", err.Error(), index, stepRes.Error())
			} else {
				err = fmt.Errorf("{failed sequence step#%d caused by: %s}", index, stepRes.Error())
			}
		}
	}
	return err
}
