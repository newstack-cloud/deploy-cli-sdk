package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
)

// LoadOperationConfig loads the blueprint operation config (provider,
// transformer, context variable and blueprint variable values) from the deploy
// config file recorded in the "deployConfigFile" config key.
//
// The deploy config file is typically produced by a CLI's PreCommandStep
// (e.g. Celerity converts app.deploy.jsonc into .celerity/deploy-config.json
// and points deployConfigFile at it). It carries values such as the deploy
// target that provider and transformer plugins require during validation,
// change staging and deployment; without it the engine invokes plugins with an
// empty deploy target.
//
// Returns (nil, nil) when no deploy config file is configured, so callers can
// pass the result straight through to a payload's Config field.
func LoadOperationConfig(confProvider *Provider) (*types.BlueprintOperationConfig, error) {
	path, _ := confProvider.GetString("deployConfigFile")
	if path == "" {
		return nil, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading deploy config file %s: %w", path, err)
	}

	var opConfig types.BlueprintOperationConfig
	if err := json.Unmarshal(data, &opConfig); err != nil {
		return nil, fmt.Errorf("parsing deploy config file %s: %w", path, err)
	}

	return &opConfig, nil
}
