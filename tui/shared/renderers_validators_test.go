package shared

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	engineerrors "github.com/newstack-cloud/bluelink/libs/deploy-engine-client/errors"
	enginetypes "github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/stretchr/testify/suite"
)

type RenderersValidatorsTestSuite struct {
	suite.Suite
	testStyles *stylespkg.Styles
}

func TestRenderersValidatorsSuite(t *testing.T) {
	suite.Run(t, new(RenderersValidatorsTestSuite))
}

func (s *RenderersValidatorsTestSuite) SetupTest() {
	s.testStyles = stylespkg.NewStyles(lipgloss.NewRenderer(os.Stdout), stylespkg.NewBluelinkPalette())
}

func (s *RenderersValidatorsTestSuite) Test_ValidateInstanceName_empty_returns_error() {
	err := ValidateInstanceName("")
	s.Require().Error(err)
	s.Contains(err.Error(), "cannot be empty")
}

func (s *RenderersValidatorsTestSuite) Test_ValidateInstanceName_two_chars_returns_error() {
	err := ValidateInstanceName("ab")
	s.Require().Error(err)
	s.Contains(err.Error(), "at least 3 characters")
}

func (s *RenderersValidatorsTestSuite) Test_ValidateInstanceName_three_chars_returns_nil() {
	s.NoError(ValidateInstanceName("abc"))
}

func (s *RenderersValidatorsTestSuite) Test_ValidateInstanceName_128_chars_returns_nil() {
	s.NoError(ValidateInstanceName(strings.Repeat("a", 128)))
}

func (s *RenderersValidatorsTestSuite) Test_ValidateInstanceName_129_chars_returns_error() {
	err := ValidateInstanceName(strings.Repeat("a", 129))
	s.Require().Error(err)
	s.Contains(err.Error(), "at most 128 characters")
}

func (s *RenderersValidatorsTestSuite) Test_ValidateInstanceName_whitespace_padded_two_chars_returns_error() {
	// "  ab  " trims to "ab" which is too short
	err := ValidateInstanceName("  ab  ")
	s.Require().Error(err)
	s.Contains(err.Error(), "at least 3 characters")
}

func (s *RenderersValidatorsTestSuite) Test_ValidateChangesetID_empty_returns_error() {
	err := ValidateChangesetID("")
	s.Require().Error(err)
	s.Contains(err.Error(), "required")
}

func (s *RenderersValidatorsTestSuite) Test_ValidateChangesetID_whitespace_only_returns_error() {
	err := ValidateChangesetID("  ")
	s.Require().Error(err)
	s.Contains(err.Error(), "required")
}

func (s *RenderersValidatorsTestSuite) Test_ValidateChangesetID_valid_returns_nil() {
	s.NoError(ValidateChangesetID("cs-123"))
}

func (s *RenderersValidatorsTestSuite) Test_DeployErrorContext_returns_non_empty_fields() {
	ctx := DeployErrorContext()
	s.NotEmpty(ctx.OperationName)
	s.NotEmpty(ctx.FailedHeader)
	s.NotEmpty(ctx.ErrorDuringHeader)
	s.NotEmpty(ctx.IssuesPreamble)
}

func (s *RenderersValidatorsTestSuite) Test_StageErrorContext_returns_non_empty_fields() {
	ctx := StageErrorContext()
	s.NotEmpty(ctx.OperationName)
	s.NotEmpty(ctx.FailedHeader)
	s.NotEmpty(ctx.ErrorDuringHeader)
	s.NotEmpty(ctx.IssuesPreamble)
}

func (s *RenderersValidatorsTestSuite) Test_DestroyErrorContext_returns_non_empty_fields() {
	ctx := DestroyErrorContext()
	s.NotEmpty(ctx.OperationName)
	s.NotEmpty(ctx.FailedHeader)
	s.NotEmpty(ctx.ErrorDuringHeader)
	s.NotEmpty(ctx.IssuesPreamble)
}

func (s *RenderersValidatorsTestSuite) Test_ErrorContext_factories_return_distinct_values() {
	deploy := DeployErrorContext()
	stage := StageErrorContext()
	destroy := DestroyErrorContext()
	s.NotEqual(deploy.OperationName, stage.OperationName)
	s.NotEqual(deploy.OperationName, destroy.OperationName)
	s.NotEqual(stage.OperationName, destroy.OperationName)
}

func (s *RenderersValidatorsTestSuite) Test_RenderErrorFooter_contains_press_q_quit() {
	result := RenderErrorFooter(s.testStyles)
	s.Contains(result, "Press")
	s.Contains(result, "q")
	s.Contains(result, "quit")
}

func (s *RenderersValidatorsTestSuite) Test_RenderDiagnostic_error_level_contains_ERROR_and_message() {
	diag := &core.Diagnostic{
		Level:   core.DiagnosticLevelError,
		Message: "something went wrong",
	}
	result := RenderDiagnostic(diag, s.testStyles)
	s.Contains(result, "ERROR")
	s.Contains(result, "something went wrong")
}

func (s *RenderersValidatorsTestSuite) Test_RenderDiagnostic_with_range_contains_line_col() {
	diag := &core.Diagnostic{
		Level:   core.DiagnosticLevelError,
		Message: "bad value",
		Range: &core.DiagnosticRange{
			Start: &source.Meta{Position: source.Position{Line: 5, Column: 10}},
		},
	}
	result := RenderDiagnostic(diag, s.testStyles)
	s.Contains(result, "5")
	s.Contains(result, "10")
}

func (s *RenderersValidatorsTestSuite) Test_RenderDiagnostic_nil_range_does_not_panic() {
	diag := &core.Diagnostic{
		Level:   core.DiagnosticLevelWarning,
		Message: "a warning",
		Range:   nil,
	}
	s.NotPanics(func() {
		result := RenderDiagnostic(diag, s.testStyles)
		s.NotContains(result, "line")
	})
}

func (s *RenderersValidatorsTestSuite) Test_RenderValidationError_with_validation_errors_contains_location_and_message() {
	clientErr := &engineerrors.ClientError{
		StatusCode: 422,
		Message:    "validation failed",
		ValidationErrors: []*engineerrors.ValidationError{
			{Location: "spec.name", Message: "must not be blank"},
		},
	}
	result := RenderValidationError(clientErr, DeployErrorContext(), s.testStyles)
	s.Contains(result, "spec.name")
	s.Contains(result, "must not be blank")
}

func (s *RenderersValidatorsTestSuite) Test_RenderValidationError_with_diagnostics_contains_Diagnostics_header() {
	diag := &core.Diagnostic{
		Level:   core.DiagnosticLevelError,
		Message: "blueprint parse error",
	}
	clientErr := &engineerrors.ClientError{
		StatusCode:            422,
		Message:               "blueprint invalid",
		ValidationDiagnostics: []*core.Diagnostic{diag},
	}
	result := RenderValidationError(clientErr, StageErrorContext(), s.testStyles)
	s.Contains(result, "Diagnostics")
}

func (s *RenderersValidatorsTestSuite) Test_RenderValidationError_with_neither_contains_client_error_message() {
	clientErr := &engineerrors.ClientError{
		StatusCode: 400,
		Message:    "generic client error message",
	}
	result := RenderValidationError(clientErr, DeployErrorContext(), s.testStyles)
	s.Contains(result, "generic client error message")
}

func (s *RenderersValidatorsTestSuite) Test_RenderStreamError_contains_event_message() {
	streamErr := &engineerrors.StreamError{
		Event: &enginetypes.StreamErrorMessageEvent{
			Message: "stream processing failed",
		},
	}
	result := RenderStreamError(streamErr, DeployErrorContext(), s.testStyles)
	s.Contains(result, "stream processing failed")
}

func (s *RenderersValidatorsTestSuite) Test_RenderGenericError_contains_header_and_error_message() {
	err := errors.New("disk is full")
	result := RenderGenericError(err, "Operation failed", s.testStyles)
	s.Contains(result, "Operation failed")
	s.Contains(result, "disk is full")
}

func (s *RenderersValidatorsTestSuite) Test_RenderChangesetTypeMismatchError_destroy_changeset_contains_correct_messages() {
	params := ChangesetTypeMismatchParams{
		IsDestroyChangeset: true,
		InstanceName:       "my-app",
		ChangesetID:        "cs-abc",
	}
	result := RenderChangesetTypeMismatchError(params, s.testStyles)
	s.Contains(result, "Cannot deploy using a destroy changeset")
	s.Contains(result, "destroy")
}

func (s *RenderersValidatorsTestSuite) Test_RenderChangesetTypeMismatchError_deploy_changeset_contains_correct_messages() {
	params := ChangesetTypeMismatchParams{
		IsDestroyChangeset: false,
		InstanceName:       "my-app",
		ChangesetID:        "cs-xyz",
	}
	result := RenderChangesetTypeMismatchError(params, s.testStyles)
	s.Contains(result, "Cannot destroy using a deploy changeset")
	s.Contains(result, "deploy")
}

func (s *RenderersValidatorsTestSuite) Test_RenderSectionHeader_contains_header_text_and_separator() {
	var sb strings.Builder
	RenderSectionHeader(&sb, "My Section", 80, s.testStyles)
	result := sb.String()
	s.Contains(result, "My Section")
	s.Contains(result, "─")
}

func (s *RenderersValidatorsTestSuite) Test_RenderLabelValue_contains_label_and_value() {
	var sb strings.Builder
	RenderLabelValue(&sb, "Status", "active", s.testStyles)
	result := sb.String()
	s.Contains(result, "Status")
	s.Contains(result, "active")
}

func (s *RenderersValidatorsTestSuite) Test_FindResourceStateByName_nil_instanceState_returns_nil() {
	s.Nil(FindResourceStateByName(nil, "res"))
}

func (s *RenderersValidatorsTestSuite) Test_FindResourceStateByName_found_returns_correct_state() {
	resState := &state.ResourceState{ResourceID: "rid-1"}
	instanceState := &state.InstanceState{
		ResourceIDs: map[string]string{"myRes": "rid-1"},
		Resources:   map[string]*state.ResourceState{"rid-1": resState},
	}
	result := FindResourceStateByName(instanceState, "myRes")
	s.Equal(resState, result)
}

func (s *RenderersValidatorsTestSuite) Test_FindResourceStateByName_not_found_returns_nil() {
	instanceState := &state.InstanceState{
		ResourceIDs: map[string]string{"myRes": "rid-1"},
		Resources:   map[string]*state.ResourceState{"rid-1": {}},
	}
	s.Nil(FindResourceStateByName(instanceState, "other"))
}

func (s *RenderersValidatorsTestSuite) Test_FindChildInstanceIDByPath_nil_instanceState_returns_empty() {
	s.Equal("", FindChildInstanceIDByPath(nil, "childA"))
}

func (s *RenderersValidatorsTestSuite) Test_FindChildInstanceIDByPath_empty_path_returns_empty() {
	s.Equal("", FindChildInstanceIDByPath(&state.InstanceState{}, ""))
}

func (s *RenderersValidatorsTestSuite) Test_FindChildInstanceIDByPath_direct_child_returns_instance_id() {
	childState := &state.InstanceState{InstanceID: "child-inst-1"}
	instanceState := &state.InstanceState{
		ChildBlueprints: map[string]*state.InstanceState{
			"childA": childState,
		},
	}
	s.Equal("child-inst-1", FindChildInstanceIDByPath(instanceState, "childA"))
}

func (s *RenderersValidatorsTestSuite) Test_FindChildInstanceIDByPath_nested_child_returns_nested_instance_id() {
	grandChildState := &state.InstanceState{InstanceID: "grand-inst-1"}
	childState := &state.InstanceState{
		InstanceID: "child-inst-1",
		ChildBlueprints: map[string]*state.InstanceState{
			"childB": grandChildState,
		},
	}
	instanceState := &state.InstanceState{
		ChildBlueprints: map[string]*state.InstanceState{
			"childA": childState,
		},
	}
	s.Equal("grand-inst-1", FindChildInstanceIDByPath(instanceState, "childA/childB"))
}

func (s *RenderersValidatorsTestSuite) Test_FindChildInstanceIDByPath_non_existent_returns_empty() {
	instanceState := &state.InstanceState{
		ChildBlueprints: map[string]*state.InstanceState{},
	}
	s.Equal("", FindChildInstanceIDByPath(instanceState, "nope"))
}

func (s *RenderersValidatorsTestSuite) Test_FindLinkIDByPath_nil_instanceState_returns_empty() {
	s.Equal("", FindLinkIDByPath(nil, "", "linkA"))
}

func (s *RenderersValidatorsTestSuite) Test_FindLinkIDByPath_found_returns_link_id() {
	linkState := &state.LinkState{LinkID: "link-id-1"}
	instanceState := &state.InstanceState{
		Links: map[string]*state.LinkState{
			"linkA": linkState,
		},
	}
	// path is empty (no child segments), linkName is "linkA"
	s.Equal("link-id-1", FindLinkIDByPath(instanceState, "", "linkA"))
}

func (s *RenderersValidatorsTestSuite) Test_FindResourceIDByPath_nil_instanceState_returns_empty() {
	s.Equal("", FindResourceIDByPath(nil, "", "resA"))
}

func (s *RenderersValidatorsTestSuite) Test_FindResourceIDByPath_found_returns_resource_id() {
	instanceState := &state.InstanceState{
		ResourceIDs: map[string]string{"resA": "rid-42"},
	}
	// path is empty string (no child segments), resourceName is "resA"
	s.Equal("rid-42", FindResourceIDByPath(instanceState, "", "resA"))
}

func (s *RenderersValidatorsTestSuite) Test_RenderElementSummary_all_zeros_writes_nothing() {
	var sb strings.Builder
	RenderElementSummary(&sb, ElementSummary{}, s.testStyles)
	s.Empty(sb.String())
}

func (s *RenderersValidatorsTestSuite) Test_RenderElementSummary_success_only_contains_count() {
	var sb strings.Builder
	RenderElementSummary(&sb, ElementSummary{SuccessCount: 3, SuccessLabel: "successful"}, s.testStyles)
	result := sb.String()
	s.Contains(result, "3")
	s.Contains(result, "successful")
}

func (s *RenderersValidatorsTestSuite) Test_RenderElementSummary_mixed_contains_all_counts() {
	var sb strings.Builder
	RenderElementSummary(&sb, ElementSummary{
		SuccessCount:     2,
		SuccessLabel:     "destroyed",
		FailureCount:     1,
		InterruptedCount: 1,
	}, s.testStyles)
	result := sb.String()
	s.Contains(result, "2")
	s.Contains(result, "destroyed")
	s.Contains(result, "1")
	s.Contains(result, "failure")
	s.Contains(result, "interrupted")
}

func (s *RenderersValidatorsTestSuite) Test_HandleViewportKeyMsg_q_sets_ShouldQuit() {
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}
	result := HandleViewportKeyMsg(msg, viewport.New(80, 24))
	s.True(result.ShouldQuit)
	s.False(result.ShouldClose)
}

func (s *RenderersValidatorsTestSuite) Test_HandleViewportKeyMsg_ctrl_c_sets_ShouldQuit() {
	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	result := HandleViewportKeyMsg(msg, viewport.New(80, 24))
	s.True(result.ShouldQuit)
	s.False(result.ShouldClose)
}

func (s *RenderersValidatorsTestSuite) Test_HandleViewportKeyMsg_esc_sets_ShouldClose() {
	msg := tea.KeyMsg{Type: tea.KeyEsc}
	result := HandleViewportKeyMsg(msg, viewport.New(80, 24))
	s.True(result.ShouldClose)
	s.False(result.ShouldQuit)
}

func (s *RenderersValidatorsTestSuite) Test_HandleViewportKeyMsg_custom_toggle_key_sets_ShouldClose() {
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("o")}
	result := HandleViewportKeyMsg(msg, viewport.New(80, 24), "o", "O")
	s.True(result.ShouldClose)
	s.False(result.ShouldQuit)
}

func (s *RenderersValidatorsTestSuite) Test_HandleViewportKeyMsg_other_key_sets_neither() {
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}
	result := HandleViewportKeyMsg(msg, viewport.New(80, 24))
	s.False(result.ShouldQuit)
	s.False(result.ShouldClose)
}

func (s *RenderersValidatorsTestSuite) Test_CheckExportsKeyMsg_q_returns_quit() {
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}
	s.Equal(ExportsKeyActionQuit, CheckExportsKeyMsg(msg))
}

func (s *RenderersValidatorsTestSuite) Test_CheckExportsKeyMsg_ctrl_c_returns_quit() {
	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	s.Equal(ExportsKeyActionQuit, CheckExportsKeyMsg(msg))
}

func (s *RenderersValidatorsTestSuite) Test_CheckExportsKeyMsg_esc_returns_close() {
	msg := tea.KeyMsg{Type: tea.KeyEsc}
	s.Equal(ExportsKeyActionClose, CheckExportsKeyMsg(msg))
}

func (s *RenderersValidatorsTestSuite) Test_CheckExportsKeyMsg_e_returns_close() {
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")}
	s.Equal(ExportsKeyActionClose, CheckExportsKeyMsg(msg))
}

func (s *RenderersValidatorsTestSuite) Test_CheckExportsKeyMsg_E_returns_close() {
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E")}
	s.Equal(ExportsKeyActionClose, CheckExportsKeyMsg(msg))
}

func (s *RenderersValidatorsTestSuite) Test_CheckExportsKeyMsg_other_returns_delegate() {
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")}
	s.Equal(ExportsKeyActionDelegate, CheckExportsKeyMsg(msg))
}

func (s *RenderersValidatorsTestSuite) Test_ExtractGrouping_nil_meta_returns_nil() {
	s.Nil(ExtractGrouping(nil))
}

func (s *RenderersValidatorsTestSuite) Test_ExtractGrouping_nil_annotations_returns_nil() {
	meta := &state.ResourceMetadataState{Annotations: nil}
	s.Nil(ExtractGrouping(meta))
}

func (s *RenderersValidatorsTestSuite) Test_ExtractGrouping_missing_one_annotation_returns_nil() {
	strVal := "celerity/function"
	meta := &state.ResourceMetadataState{
		Annotations: map[string]*core.MappingNode{
			AnnotationSourceAbstractName: {Scalar: core.ScalarFromString(strVal)},
			// AnnotationSourceAbstractType intentionally absent
		},
	}
	s.Nil(ExtractGrouping(meta))
}

func (s *RenderersValidatorsTestSuite) Test_ExtractGrouping_both_annotations_present_returns_resource_group() {
	meta := &state.ResourceMetadataState{
		Annotations: map[string]*core.MappingNode{
			AnnotationSourceAbstractName: {Scalar: core.ScalarFromString("myFunction")},
			AnnotationSourceAbstractType: {Scalar: core.ScalarFromString("celerity/function")},
		},
	}
	result := ExtractGrouping(meta)
	s.Require().NotNil(result)
	s.Equal("myFunction", result.GroupName)
	s.Equal("celerity/function", result.GroupType)
}

func (s *RenderersValidatorsTestSuite) Test_ExtractResourceCategory_nil_meta_returns_empty() {
	s.Equal("", ExtractResourceCategory(nil))
}

func (s *RenderersValidatorsTestSuite) Test_ExtractResourceCategory_missing_annotation_returns_empty() {
	meta := &state.ResourceMetadataState{
		Annotations: map[string]*core.MappingNode{},
	}
	s.Equal("", ExtractResourceCategory(meta))
}

func (s *RenderersValidatorsTestSuite) Test_ExtractResourceCategory_present_returns_category_string() {
	meta := &state.ResourceMetadataState{
		Annotations: map[string]*core.MappingNode{
			AnnotationResourceCategory: {Scalar: core.ScalarFromString(ResourceCategoryCodeHosting)},
		},
	}
	s.Equal(ResourceCategoryCodeHosting, ExtractResourceCategory(meta))
}
