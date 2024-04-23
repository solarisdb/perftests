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
		metrics map[string]*runner.Scalar[int64]
	}

	MetricsCfg struct {
		Cmds []MetricsCmd `yaml:"cmds" json:"cmds"`
	}

	MetricsCmd string
)

const (
	metricsAppendTOs       = "appendTOs"
	metricsQueryRecordsTOs = "queryRecordsTOs"

	MetricsRunName = "solaris.metrics"

	MetricsInit         MetricsCmd = "init"
	MetricsAppend       MetricsCmd = "append"
	MetricsQueryRecords MetricsCmd = "queryRecords"
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
	intMetrics := map[string]*runner.Scalar[int64]{}
	for _, cmd := range cfg.Cmds {
		switch cmd {
		case MetricsInit:
			intMetrics[metricsAppendTOs] = runner.NewScalar[int64]()
			intMetrics[metricsQueryRecordsTOs] = runner.NewScalar[int64]()
		case MetricsAppend:
			tos, _ := ctx.Value(metricsAppendTOs).(*runner.Scalar[int64])
			tos = tos.Copy()
			meanDur := time.Duration(int64(tos.Mean())).Round(time.Millisecond)
			sumDur := time.Duration(int64(tos.Sum())).Round(time.Millisecond)
			r.exec.Logger.Infof("Append TOs Metric: total %d, sum %s, mean %s", tos.Total(), sumDur.String(), meanDur.String())
			intMetrics[metricsAppendTOs] = tos
		case MetricsQueryRecords:
			tos, _ := ctx.Value(metricsQueryRecordsTOs).(*runner.Scalar[int64])
			tos = tos.Copy()
			meanDur := time.Duration(int64(tos.Mean())).Round(time.Millisecond)
			sumDur := time.Duration(int64(tos.Sum())).Round(time.Millisecond)
			r.exec.Logger.Infof("Query Records TOs Metric: total %d, sum %s, mean %s", tos.Total(), sumDur.String(), meanDur.String())
			intMetrics[metricsQueryRecordsTOs] = tos
		default:
			doneCh <- runner.NewStaticScenarioResult(ctx, fmt.Errorf("unknown metrics command: %s", cmd))
		}
	}
	doneCh <- &metricsScenarioResult{metrics: intMetrics}
	return
}

func (r *metricsScenarioResult) Ctx(ctx context.Context) context.Context {
	for name, val := range r.metrics {
		ctx = context.WithValue(ctx, name, val)
	}
	return ctx
}

func (r *metricsScenarioResult) Error() error {
	return nil
}
