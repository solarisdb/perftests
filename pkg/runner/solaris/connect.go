package solaris

import (
	"crypto/tls"
	"fmt"
	"strings"

	"github.com/solarisdb/perftests/pkg/model"
	"github.com/solarisdb/perftests/pkg/runner"
	"github.com/solarisdb/solaris/api/gen/solaris/v1"
	"github.com/solarisdb/solaris/golibs/errors"
	"github.com/solarisdb/solaris/golibs/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"context"
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
		Address string `yaml:"address" json:"address"`
	}

	connectScenarioResult struct {
		svc solaris.ServiceClient
	}
)

const (
	solarisClnt = "solarisClnt"
	ConnectName = "connect"
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

	conn, err := r.dial(cfg.Address)
	if err != nil {
		doneCh <- runner.NewStaticScenarioResult(ctx, fmt.Errorf("failed to dial to address %s: %w", cfg.Address, err))
		return
	}

	client := solaris.NewServiceClient(conn)
	doneCh <- &connectScenarioResult{
		client,
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

func isTls(addr string) bool {
	idx := strings.LastIndex(addr, ":")
	if idx == -1 {
		return false // insecure
	}
	s1 := strings.Trim(addr[idx+1:], " ")
	return s1 == "443"
}

func (r *connectScenarioResult) Ctx(ctx context.Context) context.Context {
	client := ctx.Value(solarisClnt)
	if client == nil {
		ctx = context.WithValue(ctx, solarisClnt, r.svc)
	}
	return ctx
}

func (r *connectScenarioResult) Error() error {
	return nil
}
