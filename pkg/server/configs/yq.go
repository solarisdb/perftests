package configs

import (
	"os"

	"github.com/mikefarah/yq/v4/pkg/yqlib"
	"gopkg.in/op/go-logging.v1"
)

func init() {
	yqlibLogger := logging.AddModuleLevel(logging.NewLogBackend(os.Stderr, "yqlib", 0))
	yqlibLogger.SetLevel(logging.ERROR, "")
	yqlib.GetLogger().SetBackend(yqlibLogger)
}

func JsonToYaml(json string) ([]byte, error) {
	encoder := yqlib.NewYamlEncoder(
		yqlib.NewDefaultYamlPreferences())
	decoder := yqlib.NewJSONDecoder()
	yamlResult, err := yqlib.NewStringEvaluator().Evaluate("", json, encoder, decoder)
	if err != nil {
		return nil, err
	}
	return []byte(yamlResult), nil
}
