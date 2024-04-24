package runner

import (
	"context"
	"fmt"

	"github.com/solarisdb/perftests/pkg/model"
	metrics2 "github.com/solarisdb/perftests/pkg/runner/metrics"
	"github.com/solarisdb/solaris/golibs/errors"
	"github.com/solarisdb/solaris/golibs/logging"
)

type (
	metricsCreate struct {
		exec *metricsCreateExecutor
		name string
	}

	metricsCreateExecutor struct {
		name     string
		Registry *Registry      `inject:""`
		Logger   logging.Logger `inject:""`
	}

	metricsCreateScenarioResult struct {
		metrics map[string]MetricValue
	}

	MetricsCreateCfg struct {
		Metrics map[MetricsType][]string `yaml:"metrics" json:"metrics"`
	}

	MetricValue struct {
		Value any         `yaml:"value" json:"value"`
		Type  MetricsType `yaml:"type" json:"type"`
	}

	MetricsType string
)

const (
	MetricsCreateRunName             = "metricsCreate"
	INT                  MetricsType = "INT"
	STRING               MetricsType = "STRING"
)

func NewMetricsCreate(exec *metricsCreateExecutor, prefix string) ScenarioRunner {
	return &metricsCreate{exec: exec, name: fmt.Sprintf("%s/%s-%d", prefix, exec.Name(), GetRunnerIndex())}
}

func NewMetricsCreateExecutor() ScenarioExecutor {
	return &metricsCreateExecutor{name: MetricsCreateRunName}
}

func (r *metricsCreateExecutor) Init(ctx context.Context) error {
	return r.Registry.Register(r)
}

func (r *metricsCreateExecutor) Name() string {
	return r.name
}

func (r *metricsCreateExecutor) New(prefix string) ScenarioRunner {
	return NewMetricsCreate(r, prefix)
}

func (r *metricsCreate) RunScenario(ctx context.Context, config *model.ScenarioConfig) <-chan ScenarioResult {
	r.exec.Logger.Debugf("Running scenario %s", r.name)
	defer r.exec.Logger.Debugf("Scenario finished %s", r.name)

	return r.run(ctx, config)
}

func (r *metricsCreate) run(ctx context.Context, config *model.ScenarioConfig) (doneCh chan ScenarioResult) {
	doneCh = make(chan ScenarioResult, 1)
	defer close(doneCh)

	if ctx.Err() != nil {
		doneCh <- NewStaticScenarioResult(ctx, fmt.Errorf("run context is closed %w", errors.ErrClosed))
		return
	}

	cfg, err := model.FromScenarioConfig[MetricsCreateCfg](config)
	if err != nil {
		doneCh <- NewStaticScenarioResult(ctx, fmt.Errorf("failed to parse scenario config %w", err))
		return
	}
	toCreateMetrics := map[string]MetricValue{}
	for mType, mNames := range cfg.Metrics {
		for _, mName := range mNames {
			switch mType {
			case INT:
				toCreateMetrics[mName] = MetricValue{Value: metrics2.NewScalar[int64](), Type: mType}
			case STRING:
				toCreateMetrics[mName] = MetricValue{Value: metrics2.NewString(), Type: mType}
			default:
				doneCh <- NewStaticScenarioResult(ctx, fmt.Errorf("unknown metrics type: %d", mType))
			}
		}
	}
	doneCh <- &metricsCreateScenarioResult{metrics: toCreateMetrics}
	return
}

func (r *metricsCreateScenarioResult) Ctx(ctx context.Context) context.Context {
	for name, val := range r.metrics {
		ctx = context.WithValue(ctx, name, val)
	}
	return ctx
}

func (r *metricsCreateScenarioResult) Error() error {
	return nil
}
