package runner

import (
	"fmt"

	"github.com/solarisdb/solaris/golibs/errors"
)

type (
	Registry struct {
		scenarios map[string]ScenarioExecutor
	}
)

func NewRegistry() *Registry {
	return &Registry{
		scenarios: map[string]ScenarioExecutor{},
	}
}

func (r *Registry) Register(sr ScenarioExecutor) error {
	if _, ok := r.scenarios[sr.Name()]; ok {
		return fmt.Errorf("scenario %s is already registered %w", sr.Name(), errors.ErrExist)
	}
	r.scenarios[sr.Name()] = sr
	return nil
}

func (r *Registry) Get(name string) (ScenarioExecutor, bool) {
	sr, ok := r.scenarios[name]
	return sr, ok
}
