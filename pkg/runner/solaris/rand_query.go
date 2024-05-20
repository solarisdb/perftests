package solaris

import (
	"fmt"
	"github.com/oklog/ulid/v2"
	"github.com/solarisdb/solaris/golibs/ulidutils"
	"math/rand"
	"time"

	"github.com/solarisdb/perftests/pkg/model"
	"github.com/solarisdb/perftests/pkg/runner"
	"github.com/solarisdb/solaris/api/gen/solaris/v1"
	"github.com/solarisdb/solaris/golibs/errors"
	"github.com/solarisdb/solaris/golibs/logging"

	"context"
)

type (
	randQueryMsgs struct {
		exec *randQueryMsgsExecutor
		name string
	}

	randQueryMsgsExecutor struct {
		name     string
		Registry *runner.Registry `inject:""`
		EnvCfg   *model.Config    `inject:""`
		Logger   logging.Logger   `inject:""`
	}

	RandQueryMsgsCfg struct {
		Step                int64  `yaml:"step" json:"step"`
		Number              int    `yaml:"number" json:"number"`
		TimeoutMetricName   string `yaml:"timeoutMetricName,omitempty" json:"timeoutMetricName,omitempty"`
		MsgsRateMetricName  string `yaml:"msgsRateMetricName,omitempty" json:"msgsRateMetricName,omitempty"`
		BytesRateMetricName string `yaml:"bytesRateMetricName,omitempty" json:"bytesRateMetricName,omitempty"`
	}
)

const RandQueryMsgsRunName = "solaris.randQueryMsgs"

func NewRandQueryMsgs(exec *randQueryMsgsExecutor, prefix string) runner.ScenarioRunner {
	return &randQueryMsgs{exec: exec, name: fmt.Sprintf("%s/%s-%d", prefix, exec.Name(), runner.GetRunnerIndex())}
}

func NewRandQueryMsgsExecutor() runner.ScenarioExecutor {
	return &randQueryMsgsExecutor{name: RandQueryMsgsRunName}
}

func (r *randQueryMsgsExecutor) Init(ctx context.Context) error {
	return r.Registry.Register(r)
}

func (r *randQueryMsgsExecutor) Name() string {
	return r.name
}

func (r *randQueryMsgsExecutor) New(prefix string) runner.ScenarioRunner {
	return NewRandQueryMsgs(r, prefix)
}

func (r *randQueryMsgs) RunScenario(ctx context.Context, config *model.ScenarioConfig) <-chan runner.ScenarioResult {
	r.exec.Logger.Debugf("Running scenario %s", r.name)
	defer r.exec.Logger.Debugf("Scenario finished %s", r.name)

	return r.run(ctx, config)
}

func (r *randQueryMsgs) run(ctx context.Context, config *model.ScenarioConfig) (doneCh chan runner.ScenarioResult) {
	doneCh = make(chan runner.ScenarioResult, 1)
	defer close(doneCh)

	if ctx.Err() != nil {
		doneCh <- runner.NewStaticScenarioResult(ctx, fmt.Errorf("run context is closed %w", errors.ErrClosed))
		return
	}

	cfg, err := model.FromScenarioConfig[SeqQueryMsgsCfg](config)
	if err != nil {
		doneCh <- runner.NewStaticScenarioResult(ctx, fmt.Errorf("failed to parse scenario config %w", err))
		return
	}
	if cfg.Step == 0 {
		cfg.Step = defaultQueryRecordsLimit
	}
	if cfg.Number <= 0 {
		doneCh <- runner.NewStaticScenarioResult(ctx, fmt.Errorf("rand read number should be greater than 0"))
		return
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

	req := &solaris.QueryRecordsRequest{
		LogIDs:        []string{log},
		Limit:         1,
		StartRecordID: "",
	}
	res, err := clnt.QueryRecords(ctx, req)
	if err != nil || len(res.Records) == 0 {
		doneCh <- runner.NewStaticScenarioResult(ctx, fmt.Errorf("failed to read first record: %w", err))
		return
	}
	maxID, _ := maxULID()
	fromID := res.Records[0].ID
	req = &solaris.QueryRecordsRequest{
		LogIDs:        []string{log},
		Limit:         1,
		StartRecordID: maxID,
		Descending:    true,
	}
	res, err = clnt.QueryRecords(ctx, req)
	if err != nil || len(res.Records) == 0 {
		doneCh <- runner.NewStaticScenarioResult(ctx, fmt.Errorf("failed to read last record: %w", err))
		return
	}
	toID := res.Records[0].ID
	i := 0
	from := ulid.Time(ulid.MustParse(fromID).Time())
	to := ulid.Time(ulid.MustParse(toID).Time())
	for {
		if i >= cfg.Number {
			break
		}
		nextID, err := randULID(from, to)
		if err != nil {
			doneCh <- runner.NewStaticScenarioResult(ctx, fmt.Errorf("failed to generate nextID: %w", err))
			return
		}
		req = &solaris.QueryRecordsRequest{
			LogIDs:        []string{log},
			Limit:         cfg.Step,
			StartRecordID: nextID,
		}
		start := time.Now()
		res, err = clnt.QueryRecords(ctx, req)
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
		i++
	}

	doneCh <- runner.NewStaticScenarioResult(ctx, nil)
	return
}

func randULID(from, to time.Time) (string, error) {
	randMillis := rand.Int63n(to.Sub(from).Milliseconds())
	next := from.Add(time.Millisecond * time.Duration(randMillis))
	randID := ulidutils.New()
	if err := randID.SetTime(ulid.Timestamp(next)); err != nil {
		return *new(string), err
	}
	return randID.String(), nil
}

func maxULID() (string, error) {
	maxID := ulidutils.New()
	if err := maxID.SetTime(ulid.MaxTime()); err != nil {
		return *new(string), err
	}
	return maxID.String(), nil
}
