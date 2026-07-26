package driftui

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type MessagesTestSuite struct {
	suite.Suite
}

func TestMessagesTestSuite(t *testing.T) {
	suite.Run(t, new(MessagesTestSuite))
}

func (s *MessagesTestSuite) Test_HintForContext_stage() {
	hint := HintForContext(DriftContextStage)
	s.Equal("Run bluelink stage --skip-drift-check to skip drift detection", hint)
}

func (s *MessagesTestSuite) Test_HintForContext_deploy_stage() {
	hint := HintForContext(DriftContextDeployStage)
	s.Equal("Run bluelink stage --skip-drift-check to skip drift detection", hint)
}

func (s *MessagesTestSuite) Test_HintForContext_deploy() {
	hint := HintForContext(DriftContextDeploy)
	s.Equal("Run bluelink deploy --force to override drift check", hint)
}

func (s *MessagesTestSuite) Test_HintForContext_destroy() {
	hint := HintForContext(DriftContextDestroy)
	s.Equal("Run bluelink destroy --force to override drift check", hint)
}

func (s *MessagesTestSuite) Test_HintForContext_unknown_returns_empty() {
	hint := HintForContext("unknown_context")
	s.Equal("", hint)
}

func (s *MessagesTestSuite) Test_HintForContext_empty_string_returns_empty() {
	hint := HintForContext("")
	s.Equal("", hint)
}

func (s *MessagesTestSuite) Test_SetCLIName_overrides_hint_branding() {
	defer SetCLIName("bluelink") // restore default for other tests
	SetCLIName("celerity")
	s.Equal(
		"Run celerity stage --skip-drift-check to skip drift detection",
		HintForContext(DriftContextStage),
	)
	s.Equal(
		"Run celerity deploy --force to override drift check",
		HintForContext(DriftContextDeploy),
	)
}

func (s *MessagesTestSuite) Test_SetCLIName_ignores_empty() {
	defer SetCLIName("bluelink")
	SetCLIName("celerity")
	SetCLIName("") // must not override the previously set name
	s.Equal(
		"Run celerity destroy --force to override drift check",
		HintForContext(DriftContextDestroy),
	)
}
