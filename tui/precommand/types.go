package precommand

import (
	"context"

	"github.com/newstack-cloud/deploy-cli-sdk/config"
)

// ProgressMsg represents a progress update from a Step.
type ProgressMsg struct {
	Phase  string
	Detail string
}

// Step is executed before stage/deploy engine interactions.
// Implementations may modify the deploy config (e.g. injecting context variables).
type Step interface {
	Run(ctx context.Context, confProvider *config.Provider, commandName string, progress chan<- ProgressMsg) error
}
