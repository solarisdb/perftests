package configs

import (
	"encoding/json"
	"fmt"
	"strings"

	"dario.cat/mergo"
	"github.com/solarisdb/perftests/pkg/model"
	"github.com/solarisdb/solaris/golibs/config"
	yml "gopkg.in/yaml.v2"
)

// LoadFromFile allows loading and merging configuration from a json file into
func LoadFromFile(file string) (model.Config, error) {
	en := config.NewEnricher(model.Config{})
	var err error
	switch {
	case strings.HasSuffix(file, ".json"):
		err = loadJsonFile(en, file)
	case strings.HasSuffix(file, ".yaml"):
		err = en.LoadFromYAMLFile(file)
	default:
		err = loadJsonFile(en, file)
		if err != nil {
			err = en.LoadFromYAMLFile(file)
		}
	}
	val := en.Value()
	if err != nil {
		return val, fmt.Errorf("cannot apply config file %s: %w", file, err)
	}
	return val, val.Verify()
}

func loadJsonFile[T any](en config.Enricher[T], path string) error {
	if err := en.LoadFromJSONFile(path); err != nil {
		return err
	}
	// files with secrets are stored as one-level json,
	// so we try to map such jsons to a config with special func
	if err := config.LoadJSONAndApply(en, path); err != nil {
		return err
	}
	return nil
}

func LoadFromEnvVars() (model.Config, error) {
	en := config.NewEnricher(model.Config{})
	en.ApplyEnvVariables("PERFTESTS", "_")
	val := en.Value()
	return val, val.Verify()
}

// Merge updates all empty properties of dest config with the properties of src config
func Merge(dest, src *model.Config) error {
	return mergo.Merge(dest, src)
}

func ToYml(c *model.Config) ([]byte, error) {
	//var cc model.Config
	//b, _ := json.Marshal(c)
	//return json.Marshal(c)
	//_ = json.Unmarshal(b, &cc)
	return yml.Marshal(c)

}

func ToJson(c *model.Config) ([]byte, error) {
	return json.Marshal(c)
}
