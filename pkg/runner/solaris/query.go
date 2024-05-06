package solaris

import (
	"fmt"
	"time"

	"github.com/solarisdb/perftests/pkg/model"
	"github.com/solarisdb/perftests/pkg/runner"
	"github.com/solarisdb/solaris/api/gen/solaris/v1"
	"github.com/solarisdb/solaris/golibs/errors"
	"github.com/solarisdb/solaris/golibs/logging"

	"context"
)

type (
	queryMsgs struct {
		exec *queryMsgsExecutor
		name string
	}

	queryMsgsExecutor struct {
		name     string
		Registry *runner.Registry `inject:""`
		EnvCfg   *model.Config    `inject:""`
		Logger   logging.Logger   `inject:""`
	}

	QueryMsgsCfg struct {
		Step                int64  `yaml:"step" json:"step"`
		Number              int    `yaml:"number" json:"number"`
		TimeoutMetricName   string `yaml:"timeoutMetricName" json:"timeoutMetricName"`
		MsgsRateMetricName  string `yaml:"msgsRateMetricName" json:"msgsRateMetricName"`
		BytesRateMetricName string `yaml:"bytesRateMetricName" json:"bytesRateMetricName"`
	}
)

const QueryMsgsRunName = "solaris.queryMsgs"
const defaultQueryRecordsLimit = 100

func NewQueryMsgs(exec *queryMsgsExecutor, prefix string) runner.ScenarioRunner {
	return &queryMsgs{exec: exec, name: fmt.Sprintf("%s/%s-%d", prefix, exec.Name(), runner.GetRunnerIndex())}
}

func NewQueryMsgsExecutor() runner.ScenarioExecutor {
	return &queryMsgsExecutor{name: QueryMsgsRunName}
}

func (r *queryMsgsExecutor) Init(ctx context.Context) error {
	return r.Registry.Register(r)
}

func (r *queryMsgsExecutor) Name() string {
	return r.name
}

func (r *queryMsgsExecutor) New(prefix string) runner.ScenarioRunner {
	return NewQueryMsgs(r, prefix)
}

func (r *queryMsgs) RunScenario(ctx context.Context, config *model.ScenarioConfig) <-chan runner.ScenarioResult {
	r.exec.Logger.Debugf("Running scenario %s", r.name)
	defer r.exec.Logger.Debugf("Scenario finished %s", r.name)

	return r.run(ctx, config)
}

func (r *queryMsgs) run(ctx context.Context, config *model.ScenarioConfig) (doneCh chan runner.ScenarioResult) {
	doneCh = make(chan runner.ScenarioResult, 1)
	defer close(doneCh)

	if ctx.Err() != nil {
		doneCh <- runner.NewStaticScenarioResult(ctx, fmt.Errorf("run context is closed %w", errors.ErrClosed))
		return
	}

	cfg, err := model.FromScenarioConfig[QueryMsgsCfg](config)
	if err != nil {
		doneCh <- runner.NewStaticScenarioResult(ctx, fmt.Errorf("failed to parse scenario config %w", err))
		return
	}
	if cfg.Step == 0 {
		cfg.Step = defaultQueryRecordsLimit
	}
	if cfg.Number == 0 {
		cfg.Number = 1
	} else if cfg.Number == -1 {
		//ok, it means unlimited
	}
	clnt, _ := ctx.Value(solarisClnt).(solaris.ServiceClient)
	if clnt == nil {
		doneCh <- runner.NewStaticScenarioResult(ctx, fmt.Errorf("solaris service not found"))
		return
	}

	log, _ := ctx.Value(solarisLog).(string)
	if len(log) == 0 {
		doneCh <- runner.NewStaticScenarioResult(ctx, fmt.Errorf("solaris log not found"))
		return
	}

	toMetric, _ := runner.GetDurationMetric(ctx, cfg.TimeoutMetricName)
	bytesInSecMetric, _ := runner.GetRateMetric(ctx, cfg.BytesRateMetricName)
	msgsInSecMetric, _ := runner.GetRateMetric(ctx, cfg.MsgsRateMetricName)

	fromID := ""
	i := 0
	for {
		if cfg.Number != -1 && i >= cfg.Number {
			break
		}
		req := &solaris.QueryRecordsRequest{
			LogIDs:        []string{log},
			Limit:         cfg.Step,
			StartRecordID: fromID,
		}
		start := time.Now()
		res, err := clnt.QueryRecords(ctx, req)
		if err != nil {
			doneCh <- runner.NewStaticScenarioResult(ctx, fmt.Errorf("failed to query records: %w", err))
			return
		}
		dur := time.Since(start)
		if toMetric != nil {
			toMetric.Add(dur.Nanoseconds())
		}
		if msgsInSecMetric != nil {
			msgsInSecMetric.Add(float64(len(res.Records)), dur)
		}
		if bytesInSecMetric != nil {
			var size int
			for _, rec := range res.Records {
				size += len(rec.Payload)
			}
			bytesInSecMetric.Add(float64(size), dur)
		}
		fromID = res.NextPageID
		if fromID == "" {
			break
		}
		i++
	}

	doneCh <- runner.NewStaticScenarioResult(ctx, nil)
	return
}
