package cluster

import (
	"context"
	"fmt"

	cluster2 "github.com/solarisdb/perftests/pkg/cluster"
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
		Await bool `yaml:"await,omitempty" json:"await,omitempty"`
	}
)

const (
	FinishName = "cluster.finish"
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

	_ = nodeClnt.Finish(ctx, "OK")

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
		for _, node := range nodes {
			res, err := node.Result(ctx)
			if err != nil {
				r.exec.Logger.Errorf("%v failed", node)
				continue
			}
			r.exec.Logger.Infof("%v result: %s", node, res)
		}
	}

	doneCh <- runner.NewStaticScenarioResult(ctx, nil)
	return
}
