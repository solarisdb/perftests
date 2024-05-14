package cluster

import (
	"context"
	"encoding/json"
	"fmt"

	cluster2 "github.com/solarisdb/perftests/pkg/cluster"
	"github.com/solarisdb/perftests/pkg/metrics"
	"github.com/solarisdb/perftests/pkg/model"
	"github.com/solarisdb/perftests/pkg/runner"
	"github.com/solarisdb/solaris/golibs/errors"
	"github.com/solarisdb/solaris/golibs/logging"
)

type (
	finish struct {
		exec *finishExecutor
		name string
	}

	finishExecutor struct {
		name     string
		Registry *runner.Registry `inject:""`
		Logger   logging.Logger   `inject:""`
	}

	FinishCfg struct {
		Metrics map[runner.MetricsType][]string `yaml:"metrics,omitempty" json:"metrics,omitempty"`
		Await   bool                            `yaml:"await,omitempty" json:"await,omitempty"`
	}

	nodeResult struct {
		Status  string                       `json:"status" yaml:"status"`
		Metrics map[string]typedMetricResult `json:"metrics,omitempty" yaml:"metrics,omitempty"`
	}

	typedMetricResult struct {
		Type   runner.MetricsType `json:"type" yaml:"type"`
		Result *metricResult      `json:"result" yaml:"result"`
	}

	metricResult struct {
		union json.RawMessage
	}
)

const (
	FinishName = "cluster.finish"
	statusOK   = "OK"
)

func NewFinish(exec *finishExecutor, prefix string) runner.ScenarioRunner {
	return &finish{exec: exec, name: fmt.Sprintf("%s/%s-%d", prefix, exec.Name(), runner.GetRunnerIndex())}
}

func NewFinishExecutor() runner.ScenarioExecutor {
	return &finishExecutor{name: FinishName}
}

func (r *finishExecutor) Init(ctx context.Context) error {
	return r.Registry.Register(r)
}

func (r *finishExecutor) Name() string {
	return r.name
}

func (r *finishExecutor) New(prefix string) runner.ScenarioRunner {
	return NewFinish(r, prefix)
}

func (r *finish) RunScenario(ctx context.Context, config *model.ScenarioConfig) <-chan runner.ScenarioResult {
	r.exec.Logger.Debugf("Running scenario %s", r.name)
	defer r.exec.Logger.Debugf("Scenario finished %s", r.name)

	return r.run(ctx, config)
}

func (r *finish) run(ctx context.Context, config *model.ScenarioConfig) (doneCh chan runner.ScenarioResult) {
	doneCh = make(chan runner.ScenarioResult, 1)
	defer close(doneCh)

	if ctx.Err() != nil {
		doneCh <- runner.NewStaticScenarioResult(ctx, fmt.Errorf("run context is closed %w", errors.ErrClosed))
		return
	}

	cfg, err := model.FromScenarioConfig[FinishCfg](config)
	if err != nil {
		doneCh <- runner.NewStaticScenarioResult(ctx, fmt.Errorf("failed to parse scenario config %w", err))
		return
	}

	nodeClnt, _ := ctx.Value(clusterNode).(cluster2.Node)
	if nodeClnt == nil {
		doneCh <- runner.NewStaticScenarioResult(ctx, fmt.Errorf("cluster node not found"))
		return
	}

	var myResult nodeResult
	myResult.Status = statusOK
	myResult.Metrics = make(map[string]typedMetricResult)
	for mType, mNames := range cfg.Metrics {
		for _, mName := range mNames {
			switch mType {
			case runner.DURATION:
				if metric, ok := runner.GetDurationMetric(ctx, mName); ok {
					metric = metric.Copy()
					mResult := metrics.GetDurationMetricResult(metric)
					var mr typedMetricResult
					_ = mr.ToDuration(mResult)
					myResult.Metrics[mName] = mr
				}
			case runner.RPS:
				if metric, ok := runner.GetRateMetric(ctx, mName); ok {
					metric = metric.Copy()
					mResult := metrics.GetRateMetricResult(metric)
					var mr typedMetricResult
					_ = mr.ToRPS(mResult)
					myResult.Metrics[mName] = mr
				}
			case runner.INT:
				if metric, ok := runner.GetIntMetric(ctx, mName); ok {
					metric = metric.Copy()
					mResult := metrics.GetIntMetricResult(metric)
					var mr typedMetricResult
					_ = mr.ToInt(mResult)
					myResult.Metrics[mName] = mr
				}
			case runner.STRING:
				if metric, ok := runner.GetStringMetric(ctx, mName); ok {
					metric = metric.Copy()
					mResult := metrics.GetStringMetricResult(metric)
					var mr typedMetricResult
					_ = mr.ToString(mResult)
					myResult.Metrics[mName] = mr
				}
			default:
				doneCh <- runner.NewStaticScenarioResult(ctx, fmt.Errorf("unknown metrics type: %s", mType))
				return
			}
		}
	}
	plResult, _ := json.Marshal(myResult)
	_ = nodeClnt.Finish(ctx, plResult)

	cluster, _ := ctx.Value(clusterClnt).(cluster2.Cluster)
	if cluster == nil {
		doneCh <- runner.NewStaticScenarioResult(ctx, fmt.Errorf("cluster not found"))
		return
	}

	if cfg.Await {
		nodes, err := cluster.Nodes(ctx)
		if err != nil {
			doneCh <- runner.NewStaticScenarioResult(ctx, fmt.Errorf("failed to read cluster nodes %w", err))
			return
		}
		results := make([]any, 0, len(nodes))
		var passed int
		allMetrics := make(map[string]any)
		r.exec.Logger.Debugf("// --------------------------------------------------")
		for _, node := range nodes {
			res, err := node.Result(ctx)
			if err != nil {
				r.exec.Logger.Errorf("%v failed to read result", node)
				continue
			}
			var result nodeResult
			results = append(results, result)
			_ = json.Unmarshal(res, &result)
			if result.Status == statusOK {
				passed++
			}
			nodeMetrics := make(map[string]any)
			for mName, tmr := range result.Metrics {
				//if _, ok := nodeMetrics[mName]; !ok {
				//	nodeMetrics[mName] = make(map[runner.MetricsType]any)
				//}
				//if _, ok := allMetrics[mName]; !ok {
				//	allMetrics[mName] = make(map[runner.MetricsType]any)
				//}
				switch tmr.Type {
				case runner.DURATION:
					nodeM, _ := tmr.Result.AsDuration()
					nodeMetrics[mName] = nodeM
					if allM, ok := allMetrics[mName]; ok {
						dAllM := allM.(metrics.DurationMetricResult)
						allMetrics[mName] = dAllM.Merge(nodeM)
					} else {
						allMetrics[mName] = nodeM
					}
				case runner.RPS:
					nodeM, _ := tmr.Result.AsRate()
					nodeMetrics[mName] = nodeM
					if allM, ok := allMetrics[mName]; ok {
						dAllM := allM.(metrics.RateMetricResult)
						allMetrics[mName] = dAllM.Merge(nodeM)
					} else {
						allMetrics[mName] = nodeM
					}
				case runner.INT:
					nodeM, _ := tmr.Result.AsInt()
					nodeMetrics[mName] = nodeM
					if allM, ok := allMetrics[mName]; ok {
						dAllM := allM.(metrics.IntMetricResult)
						allMetrics[mName] = dAllM.Merge(nodeM)
					} else {
						allMetrics[mName] = nodeM
					}
				case runner.STRING:
					nodeM, _ := tmr.Result.AsString()
					nodeMetrics[mName] = nodeM
					if allM, ok := allMetrics[mName]; ok {
						dAllM := allM.(metrics.StringMetricResult)
						allMetrics[mName] = dAllM.Merge(nodeM)
					} else {
						allMetrics[mName] = nodeM
					}
				default:
				}
			}
			r.exec.Logger.Debugf("// %s status: %s, metrics: %v", node, result.Status, nodeMetrics)
		}
		r.exec.Logger.Infof("// --------------------------------------------------")
		r.exec.Logger.Infof("// Total nodes: %d, Passed: %d, Failed: %d", len(nodes), passed, len(results)-passed)
		r.exec.Logger.Infof("// Total metrics:")
		for mName, res := range allMetrics {
			r.exec.Logger.Infof("//	- %s: %v", mName, res)
		}
		r.exec.Logger.Infof("// --------------------------------------------------")
	}

	doneCh <- runner.NewStaticScenarioResult(ctx, nil)
	return
}

func (mr metricResult) MarshalJSON() ([]byte, error) {
	b, err := mr.union.MarshalJSON()
	return b, err
}

func (mr *metricResult) UnmarshalJSON(b []byte) error {
	err := mr.union.UnmarshalJSON(b)
	return err
}

func (r *typedMetricResult) ToDuration(v metrics.DurationMetricResult) error {
	var result metricResult
	if err := result.FromDuraion(v); err != nil {
		return err
	}
	r.Type = runner.DURATION
	r.Result = &result
	return nil
}

func (r *typedMetricResult) ToRPS(v metrics.RateMetricResult) error {
	var result metricResult
	if err := result.FromRate(v); err != nil {
		return err
	}
	r.Type = runner.RPS
	r.Result = &result
	return nil
}

func (r *typedMetricResult) ToInt(v metrics.IntMetricResult) error {
	var result metricResult
	if err := result.FromInt(v); err != nil {
		return err
	}
	r.Type = runner.INT
	r.Result = &result
	return nil
}
func (r *typedMetricResult) ToString(v metrics.StringMetricResult) error {
	var result metricResult
	if err := result.FromString(v); err != nil {
		return err
	}
	r.Type = runner.STRING
	r.Result = &result
	return nil
}
func (mr *metricResult) FromDuraion(v metrics.DurationMetricResult) error {
	b, err := json.Marshal(v)
	mr.union = b
	return err
}
func (mr *metricResult) FromRate(v metrics.RateMetricResult) error {
	b, err := json.Marshal(v)
	mr.union = b
	return err
}
func (mr *metricResult) FromInt(v metrics.IntMetricResult) error {
	b, err := json.Marshal(v)
	mr.union = b
	return err
}
func (mr *metricResult) FromString(v metrics.StringMetricResult) error {
	b, err := json.Marshal(v)
	mr.union = b
	return err
}
func (mr metricResult) AsDuration() (metrics.DurationMetricResult, error) {
	var body metrics.DurationMetricResult
	err := json.Unmarshal(mr.union, &body)
	return body, err
}
func (mr metricResult) AsRate() (metrics.RateMetricResult, error) {
	var body metrics.RateMetricResult
	err := json.Unmarshal(mr.union, &body)
	return body, err
}
func (mr metricResult) AsInt() (metrics.IntMetricResult, error) {
	var body metrics.IntMetricResult
	err := json.Unmarshal(mr.union, &body)
	return body, err
}
func (mr metricResult) AsString() (metrics.StringMetricResult, error) {
	var body metrics.StringMetricResult
	err := json.Unmarshal(mr.union, &body)
	return body, err
}
