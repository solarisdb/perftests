package runner

import (
	"context"
	"fmt"

	"github.com/solarisdb/perftests/pkg/model"
	"github.com/solarisdb/solaris/golibs/logging"
)

type (
	ScenarioRunner interface {
		// RunScenario runs the scenario
		RunScenario(ctx context.Context, config *model.ScenarioConfig) <-chan ScenarioResult
	}

	ScenarioExecutor interface {
		Name() string
		New(prefix string) ScenarioRunner
	}

	ScenarioResult interface {
		Ctx(context.Context) context.Context
		Error() error
	}

	TestRunner struct {
		Tests    *model.Config  `inject:""`
		Registry *Registry      `inject:""`
		Logger   logging.Logger `inject:""`

		doneCh chan error
	}
)

const SkippedErrorsMap = "skippedErrors"

func NewTestRunner() *TestRunner {
	return &TestRunner{
		doneCh: make(chan error, 1),
	}
}

func (t *TestRunner) Init(ctx context.Context) error {
	return nil
}

func (t *TestRunner) Run(ctx context.Context) <-chan error {
	t.Logger.Infof("Start tests")
	defer close(t.doneCh)

	i := 1
	for _, test := range t.Tests.Tests {
		t.Logger.Infof("Test#%d %q started", i, test.Name)
		scenarioCfg := test.Scenario
		scRunner, ok := t.Registry.Get(scenarioCfg.Name)
		if !ok {
			t.doneCh <- fmt.Errorf("cannot find scenario runner %s", scenarioCfg.Name)
			return t.doneCh
		}
		ctx = context.WithValue(ctx, SkippedErrorsMap, map[string]error{})
		if result := <-scRunner.New("").RunScenario(ctx, test.Scenario.Config); result.Error() != nil {
			t.Logger.Errorf("Test#%d %q failed: %s", i, test.Name, result.Error().Error())
		} else {
			t.Logger.Infof("Test#%d %q passed", i, test.Name)
			resCtx := result.Ctx(ctx)
			skippedErrors := resCtx.Value(SkippedErrorsMap).(map[string]error)
			for runner, skippedError := range skippedErrors {
				t.Logger.Infof("skipped error: %s - %s", runner, skippedError.Error())
			}
		}
		i++
	}
	t.Logger.Infof("Tests ended")
	t.doneCh <- nil
	return t.doneCh
}
