package validateui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"github.com/newstack-cloud/deploy-cli-sdk/tui/shared"
)

// ValidateOptionsSelectedMsg is sent when the user completes the validate
// options form.
type ValidateOptionsSelectedMsg struct {
	TransformSpec          bool
	ValidateAfterTransform bool
}

// ValidateOptionsFormConfig holds the initial values for the validate options
// form.
type ValidateOptionsFormConfig struct {
	// InitialTransformSpec is the pre-populated transform spec value
	// (from flag/env/config).
	InitialTransformSpec bool
	// InitialValidateAfterTransform is the pre-populated value for
	// validating after transform (from flag/env/config).
	InitialValidateAfterTransform bool
}

// ValidateOptionsFormModel provides a form for configuring validation loader
// options.
type ValidateOptionsFormModel struct {
	styles *stylespkg.Styles

	form                   *huh.Form
	transformSpec          bool
	validateAfterTransform bool

	submitted bool
}

// NewValidateOptionsFormModel creates a new validate options form model.
func NewValidateOptionsFormModel(
	styles *stylespkg.Styles,
	config ValidateOptionsFormConfig,
) *ValidateOptionsFormModel {
	model := &ValidateOptionsFormModel{
		styles:                 styles,
		transformSpec:          config.InitialTransformSpec,
		validateAfterTransform: config.InitialValidateAfterTransform,
	}

	model.form = shared.NewThemedForm(styles,
		huh.NewGroup(
			huh.NewConfirm().
				Key("transformSpec").
				Title("Run Transformer Plugins").
				Description("Run transformer plugins during validation to expand abstract resources.").
				Affirmative("Yes").
				Negative("No").
				WithButtonAlignment(lipgloss.Left).
				Value(&model.transformSpec),
			huh.NewConfirm().
				Key("validateAfterTransform").
				Title("Validate After Transform").
				Description("Validate resources against the transformed blueprint shape (no effect unless transformer plugins also run).").
				Affirmative("Yes").
				Negative("No").
				WithButtonAlignment(lipgloss.Left).
				Value(&model.validateAfterTransform),
		),
	)

	return model
}

func (m *ValidateOptionsFormModel) Init() tea.Cmd {
	return m.form.Init()
}

func (m *ValidateOptionsFormModel) Update(msg tea.Msg) (*ValidateOptionsFormModel, tea.Cmd) {
	form, cmd := m.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
	}

	if m.form.State == huh.StateCompleted && !m.submitted {
		m.submitted = true
		m.transformSpec = m.form.GetBool("transformSpec")
		m.validateAfterTransform = m.form.GetBool("validateAfterTransform")
		return m, func() tea.Msg {
			return ValidateOptionsSelectedMsg{
				TransformSpec:          m.transformSpec,
				ValidateAfterTransform: m.validateAfterTransform,
			}
		}
	}

	return m, cmd
}

func (m *ValidateOptionsFormModel) View() string {
	return m.form.View()
}
