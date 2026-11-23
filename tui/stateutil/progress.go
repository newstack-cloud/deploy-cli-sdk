package stateutil

import (
	"github.com/newstack-cloud/deploy-cli-sdk/stateio"
	sharedui "github.com/newstack-cloud/deploy-cli-sdk/ui"
)

// DescribeEngineState describes the state an import or export works on, such as
// "memfile state in /path", or "state" when there is no engine config to describe.
func DescribeEngineState(engineConfig *stateio.EngineConfig) string {
	if engineConfig == nil {
		return "state"
	}
	return stateio.DescribeState(&engineConfig.State)
}

// RenderProgress renders the status line shown while an import or export runs: the
// spinner followed by the status, wrapped to fit within width.
func RenderProgress(spinnerView string, status string, width int) string {
	return "\n" + sharedui.WrapLabelled("  "+spinnerView+" ", status, width) + "\n"
}
