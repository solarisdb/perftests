package cluster

import (
	"context"
	"crypto/tls"
	"fmt"
	"math/rand/v2"
	"os"
	"strings"
	"time"

	cluster2 "github.com/solarisdb/perftests/pkg/cluster"
	solarCluster "github.com/solarisdb/perftests/pkg/cluster/solaris"
	"github.com/solarisdb/perftests/pkg/model"
	"github.com/solarisdb/perftests/pkg/runner"
	"github.com/solarisdb/solaris/api/gen/solaris/v1"
	"github.com/solarisdb/solaris/golibs/errors"
	"github.com/solarisdb/solaris/golibs/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

type (
	connect struct {
		exec *connectExecutor
		name string
	}

	connectExecutor struct {
		name     string
		Registry *runner.Registry `inject:""`
		Logger   logging.Logger   `inject:""`
	}

	ConnectCfg struct {
		Address       string `yaml:"address" json:"address"`
		EnvVarAddress string `yaml:"envVarAddress" json:"envVarAddress"`
		EnvRunID      string `yaml:"envRunID" json:"envRunID"`
	}

	connectScenarioResult struct {
		cluster cluster2.Cluster
		node    cluster2.Node
	}
)

const (
	clusterClnt = "clusterClnt"
	clusterNode = "clusterNode"
	ConnectName = "cluster.connect"
)

func NewConnect(exec *connectExecutor, prefix string) runner.ScenarioRunner {
	return &connect{exec: exec, name: fmt.Sprintf("%s/%s-%d", prefix, exec.Name(), runner.GetRunnerIndex())}
}

func NewConnectExecutor() runner.ScenarioExecutor {
	return &connectExecutor{name: ConnectName}
}

func (r *connectExecutor) Init(ctx context.Context) error {
	return r.Registry.Register(r)
}

func (r *connectExecutor) Name() string {
	return r.name
}

func (r *connectExecutor) New(prefix string) runner.ScenarioRunner {
	return NewConnect(r, prefix)
}

func (r *connect) RunScenario(ctx context.Context, config *model.ScenarioConfig) <-chan runner.ScenarioResult {
	r.exec.Logger.Debugf("Running scenario %s", r.name)
	defer r.exec.Logger.Debugf("Scenario finished %s", r.name)

	return r.run(ctx, config)
}

func (r *connect) run(ctx context.Context, config *model.ScenarioConfig) (doneCh chan runner.ScenarioResult) {
	doneCh = make(chan runner.ScenarioResult, 1)
	defer close(doneCh)

	if ctx.Err() != nil {
		doneCh <- runner.NewStaticScenarioResult(ctx, fmt.Errorf("run context is closed %w", errors.ErrClosed))
		return
	}

	cfg, err := model.FromScenarioConfig[ConnectCfg](config)
	if err != nil {
		doneCh <- runner.NewStaticScenarioResult(ctx, fmt.Errorf("failed to parse scenario config %w", err))
		return
	}

	address := cfg.Address
	if len(cfg.EnvVarAddress) > 0 {
		if res, ok := readEnvStr(cfg.EnvVarAddress); ok {
			address = res
		}
	}

	var runID string
	if res, ok := readEnvStr(cfg.EnvRunID); ok {
		runID = res
	} else {
		doneCh <- runner.NewStaticScenarioResult(ctx, fmt.Errorf("RunID not found by env var: %s", cfg.EnvRunID))
		return
	}

	conn, err := r.dial(address)
	if err != nil {
		doneCh <- runner.NewStaticScenarioResult(ctx, fmt.Errorf("failed to dial to address %s: %w", address, err))
		return
	}

	time.Sleep(time.Millisecond * time.Duration(rand.IntN(3000)))
	cluster, err := solarCluster.NewCluster(ctx, runID, solaris.NewServiceClient(conn))
	if err != nil {
		doneCh <- runner.NewStaticScenarioResult(ctx, fmt.Errorf("failed to create cluster %v: %w", cluster, err))
		return
	}
	node, err := cluster.AddNode(ctx)
	if err != nil {
		doneCh <- runner.NewStaticScenarioResult(ctx, fmt.Errorf("failed to add node to cluster %v: %w", cluster, err))
		return
	}
	doneCh <- &connectScenarioResult{
		cluster: cluster,
		node:    node,
	}
	return
}

func (r *connect) dial(addr string) (grpc.ClientConnInterface, error) {
	initOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`),
	}
	if isTls(addr) {
		// overwriting the TransportCredentials by TLS with default config
		initOpts[0] = grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{}))
	}
	return grpc.Dial(addr, initOpts...)
}

func readEnvStr(envVar string) (string, bool) {
	var result string
	rawEnv := os.Environ()
	for _, v := range rawEnv {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) != 2 {
			continue
		}
		if parts[0] == envVar {
			return parts[1], true
		}
	}
	return result, false
}

func isTls(addr string) bool {
	idx := strings.LastIndex(addr, ":")
	if idx == -1 {
		return false // insecure
	}
	s1 := strings.Trim(addr[idx+1:], " ")
	return s1 == "443"
}

func (r *connectScenarioResult) Ctx(ctx context.Context) context.Context {
	client := ctx.Value(clusterClnt)
	if client == nil {
		ctx = context.WithValue(ctx, clusterClnt, r.cluster)
	}
	node := ctx.Value(clusterNode)
	if node == nil {
		ctx = context.WithValue(ctx, clusterNode, r.node)
	}
	return ctx
}

func (r *connectScenarioResult) Error() error {
	return nil
}
