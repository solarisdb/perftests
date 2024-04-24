package runner

import (
	"context"
	"fmt"
	"time"

	"github.com/solarisdb/perftests/pkg/model"
	metrics2 "github.com/solarisdb/perftests/pkg/runner/metrics"
	"github.com/solarisdb/solaris/golibs/errors"
	"github.com/solarisdb/solaris/golibs/logging"
)

type (
	metricsFix struct {
		exec *metricsFixExecutor
		name string
	}

	metricsFixExecutor struct {
		name     string
		Registry *Registry      `inject:""`
		Logger   logging.Logger `inject:""`
	}

	metricsFixScenarioResult struct {
		metrics map[string]MetricValue
	}

	MetricsFixCfg struct {
		Metrics []string `yaml:"metrics" json:"metrics"`
	}
)

const (
	MetricsFixRunName = "metricsFix"
)

func NewMetricsFix(exec *metricsFixExecutor, prefix string) ScenarioRunner {
	return &metricsFix{exec: exec, name: fmt.Sprintf("%s/%s-%d", prefix, exec.Name(), GetRunnerIndex())}
}

func NewMetricsFixExecutor() ScenarioExecutor {
	return &metricsFixExecutor{name: MetricsFixRunName}
}

func (r *metricsFixExecutor) Init(ctx context.Context) error {
	return r.Registry.Register(r)
}

func (r *metricsFixExecutor) Name() string {
	return r.name
}

func (r *metricsFixExecutor) New(prefix string) ScenarioRunner {
	return NewMetricsFix(r, prefix)
}

func (r *metricsFix) RunScenario(ctx context.Context, config *model.ScenarioConfig) <-chan ScenarioResult {
	r.exec.Logger.Debugf("Running scenario %s", r.name)
	defer r.exec.Logger.Debugf("Scenario finished %s", r.name)

	return r.run(ctx, config)
}

func (r *metricsFix) run(ctx context.Context, config *model.ScenarioConfig) (doneCh chan ScenarioResult) {
	doneCh = make(chan ScenarioResult, 1)
	defer close(doneCh)

	if ctx.Err() != nil {
		doneCh <- NewStaticScenarioResult(ctx, fmt.Errorf("run context is closed %w", errors.ErrClosed))
		return
	}

	cfg, err := model.FromScenarioConfig[MetricsFixCfg](config)
	if err != nil {
		doneCh <- NewStaticScenarioResult(ctx, fmt.Errorf("failed to parse scenario config %w", err))
		return
	}
	result := map[string]MetricValue{}
	for _, mName := range cfg.Metrics {
		mValue, _ := ctx.Value(mName).(MetricValue)
		switch mValue.Type {
		case INT:
			intMetric := mValue.Value.(*metrics2.Scalar[int64]).Copy()
			meanDur := time.Duration(int64(intMetric.Mean())).Round(time.Millisecond)
			sumDur := time.Duration(int64(intMetric.Sum())).Round(time.Millisecond)
			r.exec.Logger.Infof("Metric %q: total %d, sum %s, mean %s", mName, intMetric.Total(), sumDur.String(), meanDur.String())
			result[mName] = MetricValue{Value: intMetric, Type: mValue.Type}
		case STRING:
			strMetric := mValue.Value.(*metrics2.String).Copy()
			r.exec.Logger.Infof("Metric %q: total %d, value %s", mName, strMetric.Total(), strMetric.String())
			result[mName] = MetricValue{Value: strMetric, Type: mValue.Type}
		default:
			doneCh <- NewStaticScenarioResult(ctx, fmt.Errorf("unknown metrics type: %d", mValue.Type))
		}
	}
	doneCh <- &metricsFixScenarioResult{metrics: result}
	return
}

func (r *metricsFixScenarioResult) Ctx(ctx context.Context) context.Context {
	for name, val := range r.metrics {
		ctx = context.WithValue(ctx, name, val)
	}
	return ctx
}

func (r *metricsFixScenarioResult) Error() error {
	return nil
}
