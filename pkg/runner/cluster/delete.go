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
	deleteCluster struct {
		exec *deleteClusterExecutor
		name string
	}

	deleteClusterExecutor struct {
		name     string
		Registry *runner.Registry `inject:""`
		Logger   logging.Logger   `inject:""`
	}

	deleteClusterScenarioResult struct {
		logID string
	}
)

const (
	DeleteClusterName = "cluster.delete"
)

func NewDeleteCluster(exec *deleteClusterExecutor, prefix string) runner.ScenarioRunner {
	return &deleteCluster{exec: exec, name: fmt.Sprintf("%s/%s-%d", prefix, exec.Name(), runner.GetRunnerIndex())}
}

func NewDeleteClusterExecutor() runner.ScenarioExecutor {
	return &deleteClusterExecutor{name: DeleteClusterName}
}

func (r *deleteClusterExecutor) Init(ctx context.Context) error {
	return r.Registry.Register(r)
}

func (r *deleteClusterExecutor) Name() string {
	return r.name
}

func (r *deleteClusterExecutor) New(prefix string) runner.ScenarioRunner {
	return NewDeleteCluster(r, prefix)
}

func (r *deleteCluster) RunScenario(ctx context.Context, config *model.ScenarioConfig) <-chan runner.ScenarioResult {
	r.exec.Logger.Debugf("Running scenario %s", r.name)
	defer r.exec.Logger.Debugf("Scenario finished %s", r.name)

	return r.run(ctx, config)
}

func (r *deleteCluster) run(ctx context.Context, _ *model.ScenarioConfig) (doneCh chan runner.ScenarioResult) {
	doneCh = make(chan runner.ScenarioResult, 1)
	defer close(doneCh)

	if ctx.Err() != nil {
		doneCh <- runner.NewStaticScenarioResult(ctx, fmt.Errorf("run context is closed %w", errors.ErrClosed))
		return
	}

	cluster, _ := ctx.Value(clusterClnt).(cluster2.Cluster)
	if cluster == nil {
		doneCh <- runner.NewStaticScenarioResult(ctx, fmt.Errorf("cluster not found"))
		return
	}
	err := cluster.Delete(ctx)
	if err != nil {
		doneCh <- runner.NewStaticScenarioResult(ctx, fmt.Errorf("failed to delete cluster %s: %w", cluster, err))
		return
	}
	doneCh <- &deleteClusterScenarioResult{}
	return
}

func (r *deleteClusterScenarioResult) Ctx(ctx context.Context) context.Context {
	return context.WithValue(ctx, clusterClnt, nil)
}

func (r *deleteClusterScenarioResult) Error() error {
	return nil
}
