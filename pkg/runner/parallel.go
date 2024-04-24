package runner

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/solarisdb/perftests/pkg/model"
	"github.com/solarisdb/solaris/golibs/container"
	context2 "github.com/solarisdb/solaris/golibs/context"
	"github.com/solarisdb/solaris/golibs/errors"
	"github.com/solarisdb/solaris/golibs/logging"
)

type (
	ParallelRunner struct {
		exec *parallelExecutor
		name string

		wg        *sync.WaitGroup
		lock      sync.Mutex
		stepRslts map[int]ScenarioResult
		runCtx    context.Context
		resultCh  chan ScenarioResult
		doneCh    chan struct{}
		doneCtx   context.Context
		stepCount int32
	}
	parallelExecutor struct {
		name     string
		Registry *Registry      `inject:""`
		Logger   logging.Logger `inject:""`
	}
	ParallelCfg struct {
		SkipErrors bool             `yaml:"skipErrors,omitempty" json:"skipErrors,omitempty"`
		Steps      []model.Scenario `yaml:"steps" json:"steps"`
	}

	parallelScenarioResult struct {
		results map[int]ScenarioResult
		runner  *ParallelRunner

		skipErrors bool
	}
)

const ParallelRunName = "parallel"

func NewParallelRunner(exec *parallelExecutor, prefix string) ScenarioRunner {
	doneCh := make(chan struct{})
	return &ParallelRunner{
		exec:      exec,
		wg:        &sync.WaitGroup{},
		stepRslts: make(map[int]ScenarioResult),
		resultCh:  make(chan ScenarioResult, 1),
		doneCh:    doneCh,
		doneCtx:   context2.WrapChannel(doneCh),
		stepCount: 0,
		name:      fmt.Sprintf("%s/%s-%d", prefix, exec.Name(), GetRunnerIndex()),
	}
}

func NewParallelExecutor() ScenarioExecutor {
	return &parallelExecutor{name: ParallelRunName}
}

func (r *parallelExecutor) Init(ctx context.Context) error {
	return r.Registry.Register(r)
}

func (r *parallelExecutor) Name() string {
	return r.name
}

func (r *parallelExecutor) New(prefix string) ScenarioRunner {
	return NewParallelRunner(r, prefix)
}

func (r *ParallelRunner) RunScenario(ctx context.Context, config *model.ScenarioConfig) <-chan ScenarioResult {
	r.exec.Logger.Debugf("Running scenario %s", r.name)
	//defer r.exec.Logger.Debugf("Scenario finished %s", r.name)

	return r.run(ctx, config)
}

func (r *ParallelRunner) run(ctx context.Context, config *model.ScenarioConfig) (resultCh chan ScenarioResult) {
	r.runCtx = ctx
	resultCh = r.resultCh

	if ctx.Err() != nil {
		r.resultCh <- &staticScenarioResult{ctx, fmt.Errorf("run context is closed %w", errors.ErrClosed)}
		close(r.doneCh)
		close(r.resultCh)
		r.exec.Logger.Debugf("Scenario finished %s", r.name)
		return
	}

	cfg, err := model.FromScenarioConfig[ParallelCfg](config)
	if err != nil {
		r.resultCh <- &staticScenarioResult{ctx, fmt.Errorf("failed to parse scenario config %w", err)}
		close(r.doneCh)
		close(r.resultCh)
		r.exec.Logger.Debugf("Scenario finished %s", r.name)
		return
	}
	r.wg.Add(1)
	go func() {
		defer func() {
			defer close(r.resultCh)
			close(r.doneCh)
			r.wg.Wait() // await steps were added between wait and close
			r.exec.Logger.Debugf("Scenario finished %s", r.name)
		}()
		r.wg.Wait()
		r.lock.Lock()
		results := container.CopyMap(r.stepRslts)
		r.lock.Unlock()
		r.resultCh <- &parallelScenarioResult{results: results, runner: r, skipErrors: cfg.SkipErrors}
	}()
	defer r.wg.Done()

	for _, step := range cfg.Steps {
		inStep := step
		indx := int(atomic.AddInt32(&r.stepCount, 1))
		r.startStep(r.runCtx, indx, &inStep, false)
	}
	return
}

func (r *ParallelRunner) startStep(runCtx context.Context, index int, pStep *model.Scenario, await bool) <-chan ScenarioResult {
	r.wg.Add(1)
	var doneCh chan ScenarioResult
	if await {
		doneCh = make(chan ScenarioResult, 1)
	}
	go func() {
		if await {
			defer func() {
				r.lock.Lock()
				stepRes := r.stepRslts[index]
				r.lock.Unlock()
				doneCh <- stepRes
				close(doneCh)
			}()
		}
		defer r.wg.Done()
		stepRunner, ok := r.exec.Registry.Get(pStep.Name)
		if !ok {
			r.lock.Lock()
			r.stepRslts[index] = &staticScenarioResult{runCtx, fmt.Errorf("failed to get runner for step %s: %w", pStep.Name, errors.ErrNotExist)}
			r.lock.Unlock()
			return
		}

		result := <-stepRunner.New(r.name).RunScenario(runCtx, pStep.Config)
		r.lock.Lock()
		r.stepRslts[index] = result
		r.lock.Unlock()
	}()
	return doneCh
}

func (r *ParallelRunner) addStep() (int, bool) {
	if r.doneCtx.Err() == nil {
		indx := int(atomic.AddInt32(&r.stepCount, 1))
		return indx, true
	}
	return 0, false
}

func (r *ParallelRunner) AddScenario(runCtx context.Context, step model.Scenario) (bool, <-chan ScenarioResult) {
	if indx, ok := r.addStep(); ok {
		doneCh := r.startStep(runCtx, indx, &step, true)
		return true, doneCh
	}
	return false, nil
}

func (r *parallelScenarioResult) Ctx(ctx context.Context) context.Context {
	for _, stepRes := range r.results {
		ctx = stepRes.Ctx(ctx)
	}
	if r.skipErrors {
		eList := container.CopyMap(ctx.Value(SkippedErrorsMap).(map[string]error))
		for index, stepRes := range r.results {
			if stepErr := stepRes.Error(); stepErr != nil {
				eList[fmt.Sprintf("%s/step[%d]", r.runner.name, index)] = stepErr
			}
		}
		ctx = context.WithValue(ctx, SkippedErrorsMap, eList)
	}
	return ctx
}

func (r *parallelScenarioResult) Error() error {
	var err error
	if r.skipErrors {
		return nil
	}
	for index, stepRes := range r.results {
		if stepRes.Error() != nil {
			if err != nil {
				err = fmt.Errorf("%s,\n{failed parallel step index[%d] caused by: %s}", err.Error(), index, stepRes.Error())
			} else {
				err = fmt.Errorf("{failed parallel step index[%d] caused by: %s}", index, stepRes.Error())
			}
		}
	}
	return err
}
