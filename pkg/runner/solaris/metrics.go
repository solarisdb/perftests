package solaris

import (
	"context"
	"fmt"
	"time"

	"github.com/solarisdb/perftests/pkg/model"
	"github.com/solarisdb/perftests/pkg/runner"
	"github.com/solarisdb/solaris/golibs/errors"
	"github.com/solarisdb/solaris/golibs/logging"
)

type (
	metrics struct {
		exec *metricsExecutor
		name string
	}

	metricsExecutor struct {
		name     string
		Registry *runner.Registry `inject:""`
		Logger   logging.Logger   `inject:""`

		replyTOs *runner.Scalar[int64]
	}

	metricsScenarioResult struct {
		metricValue *runner.Scalar[int64]
	}

	MetricsCfg struct {
		Cmd MetricsCmd `yaml:"cmd" json:"cmd"`
	}

	MetricsCmd string
)

const (
	metricsAppendTOs = "appendTOs"

	MetricsRunName = "metrics"

	MetricsInit   MetricsCmd = "init"
	MetricsAppend MetricsCmd = "append"
)

func NewMetrics(exec *metricsExecutor, prefix string) runner.ScenarioRunner {
	return &metrics{exec: exec, name: fmt.Sprintf("%s/%s-%d", prefix, exec.Name(), runner.GetRunnerIndex())}
}

func NewMetricsExecutor() runner.ScenarioExecutor {
	return &metricsExecutor{name: MetricsRunName}
}

func (r *metricsExecutor) Init(ctx context.Context) error {
	return r.Registry.Register(r)
}

func (r *metricsExecutor) Name() string {
	return r.name
}

func (r *metricsExecutor) New(prefix string) runner.ScenarioRunner {
	return NewMetrics(r, prefix)
}

func (r *metrics) RunScenario(ctx context.Context, config *model.ScenarioConfig) <-chan runner.ScenarioResult {
	r.exec.Logger.Debugf("Running scenario %s", r.name)
	defer r.exec.Logger.Debugf("Scenario finished %s", r.name)

	return r.run(ctx, config)
}

func (r *metrics) run(ctx context.Context, config *model.ScenarioConfig) (doneCh chan runner.ScenarioResult) {
	doneCh = make(chan runner.ScenarioResult, 1)
	defer close(doneCh)

	if ctx.Err() != nil {
		doneCh <- runner.NewStaticScenarioResult(ctx, fmt.Errorf("run context is closed %w", errors.ErrClosed))
		return
	}

	cfg, err := model.FromScenarioConfig[MetricsCfg](config)
	if err != nil {
		doneCh <- runner.NewStaticScenarioResult(ctx, fmt.Errorf("failed to parse scenario config %w", err))
		return
	}
	switch cfg.Cmd {
	case MetricsInit:
		doneCh <- &metricsScenarioResult{runner.NewScalar[int64]()}
	case MetricsAppend:
		tos, _ := ctx.Value(metricsAppendTOs).(*runner.Scalar[int64])
		tos = tos.Copy()
		meanDur := time.Duration(int64(tos.Mean())).Round(time.Millisecond)
		r.exec.Logger.Infof("Append TOs Metric: total %d, mean %s", tos.Total(), meanDur.String())
		doneCh <- &metricsScenarioResult{tos}
	default:
		doneCh <- runner.NewStaticScenarioResult(ctx, fmt.Errorf("unknown metrics command: %s", cfg.Cmd))
	}
	return
}

func (r *metricsScenarioResult) Ctx(ctx context.Context) context.Context {
	ctx = context.WithValue(ctx, metricsAppendTOs, r.metricValue)
	return ctx
}

func (r *metricsScenarioResult) Error() error {
	return nil
}
