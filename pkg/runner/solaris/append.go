package solaris

import (
	"fmt"
	"time"

	"github.com/solarisdb/perftests/pkg/model"
	"github.com/solarisdb/perftests/pkg/runner"
	"github.com/solarisdb/solaris/api/gen/solaris/v1"
	"github.com/solarisdb/solaris/golibs/container"
	"github.com/solarisdb/solaris/golibs/errors"
	"github.com/solarisdb/solaris/golibs/logging"

	"context"
)

type (
	appendMsg struct {
		exec *appendMsgExecutor
		name string
	}

	appendMsgExecutor struct {
		name     string
		Registry *runner.Registry `inject:""`
		EnvCfg   *model.Config    `inject:""`
		Logger   logging.Logger   `inject:""`
	}

	AppendCfg struct {
		MessageSize int `yaml:"messageSize" json:"messageSize"`
		BatchSize   int `yaml:"batchSize" json:"batchSize"`
	}
)

const AppendRunName = "solarisAppend"

func NewAppendMsg(exec *appendMsgExecutor, prefix string) runner.ScenarioRunner {
	return &appendMsg{exec: exec, name: fmt.Sprintf("%s/%s-%d", prefix, exec.Name(), runner.GetRunnerIndex())}
}

func NewAppendMsgExecutor() runner.ScenarioExecutor {
	return &appendMsgExecutor{name: AppendRunName}
}

func (r *appendMsgExecutor) Init(ctx context.Context) error {
	return r.Registry.Register(r)
}

func (r *appendMsgExecutor) Name() string {
	return r.name
}

func (r *appendMsgExecutor) New(prefix string) runner.ScenarioRunner {
	return NewAppendMsg(r, prefix)
}

func (r *appendMsg) RunScenario(ctx context.Context, config *model.ScenarioConfig) <-chan runner.ScenarioResult {
	r.exec.Logger.Debugf("Running scenario %s", r.name)
	defer r.exec.Logger.Debugf("Scenario finished %s", r.name)

	return r.run(ctx, config)
}

func (r *appendMsg) run(ctx context.Context, config *model.ScenarioConfig) (doneCh chan runner.ScenarioResult) {
	doneCh = make(chan runner.ScenarioResult, 1)
	defer close(doneCh)

	if ctx.Err() != nil {
		doneCh <- runner.NewStaticScenarioResult(ctx, fmt.Errorf("run context is closed %w", errors.ErrClosed))
		return
	}

	cfg, err := model.FromScenarioConfig[AppendCfg](config)
	if err != nil {
		doneCh <- runner.NewStaticScenarioResult(ctx, fmt.Errorf("failed to parse scenario config %w", err))
		return
	}
	if cfg.BatchSize == 0 {
		cfg.BatchSize = 1
	}

	//prepareMessage
	payl := make([]byte, cfg.MessageSize, cfg.MessageSize)
	container.SliceFill(payl, 'z')
	records := make([]*solaris.Record, cfg.BatchSize, cfg.BatchSize)
	container.SliceFill(records, &solaris.Record{Payload: payl})

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

	appendTOs, _ := ctx.Value(metricsAppendTOs).(*runner.Scalar[int64])

	req := &solaris.AppendRecordsRequest{
		LogID:   log,
		Records: records,
	}
	start := time.Now()
	_, err = clnt.AppendRecords(ctx, req)
	if err != nil {
		doneCh <- runner.NewStaticScenarioResult(ctx, fmt.Errorf("failed to append records: %w", err))
		return
	}
	if appendTOs != nil {
		appendTOs.Add(time.Since(start).Nanoseconds())
	}

	doneCh <- runner.NewStaticScenarioResult(ctx, nil)
	return
}
