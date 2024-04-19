package model

import (
	"encoding/json"
	"fmt"
	"regexp"

	yml "gopkg.in/yaml.v2"
)

type (
	// Config defines all the configuration parameters of the testopia
	Config struct {
		Log LoggingConfig `yaml:"log" mapstructure:"log" json:"log"`

		Tests map[string]Test `yaml:"tests"  json:"tests"`
	}

	Test struct {
		Name     string   `yaml:"name" json:"name"`
		Scenario Scenario `yaml:"scenario" json:"scenario"`
	}

	Scenario struct {
		Name   string          `yaml:"name" json:"name"`
		Config *ScenarioConfig `yaml:"config" json:"config"`
	}

	ScenarioConfig struct {
		RawCfg json.RawMessage
	}

	LoggingConfig struct {
		// Level describes desired logging level
		Level string `yaml:"level" json:"level"`
	}
)

func (a *Config) Verify() error {
	return nil
}

func (a *Config) String() string {
	b, err := yml.Marshal(a)
	if err != nil {
		return fmt.Sprintf("Can't convert to string: %v", err)
	}
	password := regexp.MustCompile(`(?i)password(.*)`)
	creds := regexp.MustCompile(`(?i)(creds|credentials)(.*)`)
	res := password.ReplaceAllString(string(b), "password: ***redacted***")
	res = creds.ReplaceAllString(res, "credentials: ***redacted***")
	return res
}

func FromScenarioConfig[T any](cc *ScenarioConfig) (T, error) {
	var v T
	err := json.Unmarshal(cc.RawCfg, &v)
	return v, err
}

func ToScenarioConfig(t any) *ScenarioConfig {
	var sc ScenarioConfig
	sc.RawCfg, _ = json.Marshal(t)
	return &sc
}

func (cc ScenarioConfig) MarshalJSON() ([]byte, error) {
	b, err := cc.RawCfg.MarshalJSON()
	return b, err
}

func (cc *ScenarioConfig) UnmarshalJSON(b []byte) error {
	err := cc.RawCfg.UnmarshalJSON(b)
	return err
}
