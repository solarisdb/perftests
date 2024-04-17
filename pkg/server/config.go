package server

import (
	"github.com/solarisdb/perftests/pkg/model"
)

func GetDefaultConfig() *model.Config {
	return &model.Config{
		Log:   model.LoggingConfig{Level: "trace"},
		Tests: []model.Test{},
	}
}
