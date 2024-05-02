package runner

import (
	"context"
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"time"

	"github.com/solarisdb/perftests/pkg/model"
	"github.com/solarisdb/solaris/golibs/errors"
	"github.com/solarisdb/solaris/golibs/logging"
)

type (
	delayRunner struct {
		exec *delayExecutor
		name string
	}
	delayExecutor struct {
		name     string
		Registry *Registry      `inject:""`
		Logger   logging.Logger `inject:""`
	}

	DelayCfg struct {
		Function string `yaml:"function" json:"function"`
	}
)

var (
	normalDelayRegexp   = regexp.MustCompile(`normal\s*\(\s*(\d+)\s*,\s*(\d+)\s*\)`)
	uniformDelayRegexp  = regexp.MustCompile(`uniform\s*\(\s*(\d+)\s*,\s*(\d+)\s*\)`)
	constantDelayRegexp = regexp.MustCompile(`constant\s*\(\s*(\d+)\s*\)`)
)

const DelayRunName = "delay"

func NewDelayRunner(exec *delayExecutor, prefix string) ScenarioRunner {
	return &delayRunner{exec: exec, name: fmt.Sprintf("%s/%s-%d", prefix, exec.Name(), GetRunnerIndex())}
}

func NewDelayExecutor() ScenarioExecutor {
	return &delayExecutor{name: DelayRunName}
}

func (r *delayExecutor) Init(ctx context.Context) error {
	return r.Registry.Register(r)
}

func (r *delayExecutor) Name() string {
	return r.name
}

func (r *delayExecutor) New(prefix string) ScenarioRunner {
	return NewDelayRunner(r, prefix)
}

func (r *delayRunner) RunScenario(ctx context.Context, config *model.ScenarioConfig) <-chan ScenarioResult {
	r.exec.Logger.Debugf("Running scenario %s", r.name)
	defer r.exec.Logger.Debugf("Scenario finished %s", r.name)

	return r.run(ctx, config)
}

func (r *delayRunner) run(ctx context.Context, config *model.ScenarioConfig) (doneCh chan ScenarioResult) {
	doneCh = make(chan ScenarioResult, 1)
	defer close(doneCh)
	if ctx.Err() != nil {
		doneCh <- &staticScenarioResult{ctx, fmt.Errorf("run context is closed %w", errors.ErrClosed)}
		return
	}
	cfg, err := model.FromScenarioConfig[DelayCfg](config)
	if err != nil {
		doneCh <- &staticScenarioResult{ctx, fmt.Errorf("failed to parse scenario config %w", err)}
		return
	}

	delay, err := r.getDelay(cfg.Function)
	if err != nil {
		doneCh <- &staticScenarioResult{ctx, fmt.Errorf("failed to parse delay function: %w", err)}
		return
	}

	time.Sleep(time.Millisecond * time.Duration(delay))

	doneCh <- &staticScenarioResult{ctx: ctx}
	return
}

func (r *delayRunner) getDelay(delayFunc string) (int64, error) {
	input := []byte(delayFunc)
	if normalDelayRegexp.Match(input) {
		params := normalDelayRegexp.FindSubmatch(input)

		mean, err := strconv.Atoi(string(params[1]))
		if err != nil {
			return 0, err
		}
		stdDev, err := strconv.Atoi(string(params[2]))
		if err != nil {
			return 0, err
		}

		return int64(rand.NormFloat64()*float64(stdDev) + float64(mean)), nil
	} else if uniformDelayRegexp.Match(input) {
		params := uniformDelayRegexp.FindSubmatch(input)

		min, err := strconv.Atoi(string(params[1]))
		if err != nil {
			return 0, err
		}
		max, err := strconv.Atoi(string(params[2]))
		if err != nil {
			return 0, err
		}
		if min > max {
			return 0, fmt.Errorf("max should be greater than min: %w", errors.ErrInvalid)
		}
		return rand.Int63n(int64(max-min)) + int64(min), nil
	} else if constantDelayRegexp.Match(input) {
		params := constantDelayRegexp.FindSubmatch(input)
		value, err := strconv.Atoi(string(params[1]))
		if err != nil {
			return 0, err
		}
		return int64(value), nil
	}

	return 0, fmt.Errorf("unknown delay function: %s: %w", delayFunc, errors.ErrInvalid)
}
