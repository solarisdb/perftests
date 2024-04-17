package runner

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/solarisdb/perftests/pkg/model"
	errors2 "github.com/solarisdb/solaris/golibs/errors"
	"github.com/solarisdb/solaris/golibs/timeout"
)

var runnerCouner int32

func GetRunnerIndex() int {
	return int(atomic.AddInt32(&runnerCouner, 1))
}

type (
	staticScenarioResult struct {
		ctx   context.Context
		error error
	}
)

func NewStaticScenarioResult(ctx context.Context, err error) ScenarioResult {
	return &staticScenarioResult{ctx: ctx, error: err}
}

func (r *staticScenarioResult) Ctx(ctx context.Context) context.Context {
	return r.ctx
}

func (r *staticScenarioResult) Error() error {
	return r.error
}

type (
	mergedContext struct {
		ch        <-chan struct{}
		main, aux context.Context
	}
)

var _ context.Context = (*mergedContext)(nil)

func MergedContext(main, aux context.Context) context.Context {
	comCh := make(chan struct{})
	go func() {
		select {
		case <-main.Done():
			close(comCh)
		case <-aux.Done():
			close(comCh)
		}
	}()
	return &mergedContext{
		ch:   comCh,
		main: main,
		aux:  aux,
	}
}

func (cc *mergedContext) Deadline() (deadline time.Time, ok bool) {
	return cc.main.Deadline()
}

func (cc *mergedContext) Done() <-chan struct{} {
	return cc.ch
}

func (cc *mergedContext) Err() error {
	select {
	case _, ok := <-cc.ch:
		if ok {
			panic("Improper use of the the context wrapper")
		}
		return fmt.Errorf("the underlying channel of mergedContext was closed: %w", errors2.ErrClosed)
	default:
		return nil
	}
}

func (cc *mergedContext) Value(key interface{}) interface{} {
	return cc.main.Value(key)
}

type StopWaiter struct {
	stopped     int32
	nextCheck   atomic.Value //time.Time
	wg          sync.WaitGroup
	stoppedByTO int32
}

func NewStopWaiter() *StopWaiter {
	sw := StopWaiter{}
	sw.nextCheck.Store(time.Now())
	sw.wg.Add(1)
	return &sw
}

func (sw *StopWaiter) Stop() bool {
	return sw.doStop(false)
}

func (sw *StopWaiter) doStop(byTimeout bool) bool {
	if atomic.CompareAndSwapInt32(&sw.stopped, 0, 1) {
		if byTimeout {
			atomic.AddInt32(&sw.stoppedByTO, 1)
		}
		sw.wg.Done()
		return true
	}
	return false
}

func (sw *StopWaiter) ExtendLife(extendTO time.Duration) bool {
	for atomic.LoadInt32(&sw.stopped) == 0 {
		currCheckTime, _ := sw.nextCheck.Load().(time.Time)
		nextCheckTime := currCheckTime.Add(extendTO)
		if sw.nextCheck.CompareAndSwap(currCheckTime, nextCheckTime) {
			checkF := func() {
				checkTime, _ := sw.nextCheck.Load().(time.Time)
				if checkTime.Before(time.Now()) {
					sw.doStop(true)
				}
			}
			timeout.Call(checkF, nextCheckTime.Sub(time.Now()))
			return true
		}
	}
	return false
}

func (sw *StopWaiter) IsAlive() bool {
	return atomic.LoadInt32(&sw.stopped) == 0
}

func (sw *StopWaiter) IsStopped() bool {
	return !sw.IsAlive()
}

func (sw *StopWaiter) IsStoppedByTimeout() bool {
	return atomic.LoadInt32(&sw.stoppedByTO) > 0
}

func (sw *StopWaiter) AwaitStop() {
	sw.wg.Wait()
}

func DelayedAction(delay time.Duration, action *model.Scenario) (delayedAction *model.Scenario) {
	if delay > 0 {
		delayedAction = &model.Scenario{
			Name: SequenceRunName,
			Config: model.ToScenarioConfig(&SequenceCfg{
				Steps: []model.Scenario{
					{
						Name:   PauseRunName,
						Config: model.ToScenarioConfig(&PauseCfg{Value: delay.String()}),
					},
					*action,
				},
			}),
		}
	} else {
		delayedAction = action
	}
	return
}
