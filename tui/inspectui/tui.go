package inspectui

import (
	"errors"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/newstack-cloud/deploy-cli-sdk/engine"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	"go.uber.org/zap"
)

type inspectSessionState uint32

const (
	inspectInstanceInput inspectSessionState = iota
	inspectLoading
	inspectViewing
)

// MainModel is the top-level model for the inspect command TUI.
type MainModel struct {
	sessionState inspectSessionState
	quitting     bool

	// Sub-models
	instanceForm *InstanceInputFormModel
	inspect      *InspectModel

	// Config from flags
	instanceID   string
	instanceName string

	// Runtime state
	headless bool
	jsonMode bool
	engine   engine.DeployEngine
	logger   *zap.Logger

	styles *stylespkg.Styles
	Error  error
}

// Init initializes the main model.
func (m MainModel) Init() tea.Cmd {
	cmds := []tea.Cmd{}

	if m.instanceForm != nil {
		cmds = append(cmds, m.instanceForm.Init())
	}

	if m.inspect != nil {
		cmds = append(cmds, m.inspect.Init())
	}

	// If we're starting in loading state (identifier already provided), fetch the instance state
	if m.sessionState == inspectLoading {
		cmds = append(cmds, fetchInstanceStateCmd(*m.inspect))
	}

	return tea.Batch(cmds...)
}

// Update handles messages for the main model.
func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case InstanceInputMsg:
		return m.handleInstanceInputMsg(msg)
	case InstanceStateFetchedMsg:
		return m.handleInstanceStateFetchedMsg(msg)
	case InstanceNotFoundMsg:
		return m.handleInstanceNotFoundMsg(msg)
	case InspectStreamStartedMsg:
		return m.handleInspectStreamStartedMsg()
	case StateRefreshTickMsg:
		return m.handleStateRefreshTickMsg()
	case InstanceStateRefreshedMsg:
		return m.handleInstanceStateRefreshedMsg(msg)
	case InspectEventMsg:
		return m.handleInspectEventMsg(msg)
	case InspectStreamClosedMsg:
		return m.handleInspectStreamClosedMsg()
	case InspectErrorMsg:
		return m.handleInspectErrorMsg(msg)
	case spinner.TickMsg:
		return m.handleSpinnerTickMsg(msg)
	case tea.WindowSizeMsg:
		return m.handleWindowSizeMsg(msg)
	case tea.KeyMsg:
		newModel, cmd, handled := m.handleKeyMsg(msg)
		if handled {
			return newModel, cmd
		}
	}

	return m.handleSessionStateRouting(msg)
}

// View renders the main model.
func (m MainModel) View() string {
	if m.quitting {
		return m.styles.Muted.Margin(1, 0, 2, 4).Render("See you next time.")
	}

	if m.Error != nil && !m.headless {
		return m.inspect.renderError(m.Error)
	}

	switch m.sessionState {
	case inspectInstanceInput:
		if m.instanceForm != nil {
			return m.instanceForm.View()
		}
		return ""
	case inspectLoading:
		return m.renderLoading()
	case inspectViewing:
		if m.inspect != nil {
			return m.inspect.View()
		}
		return ""
	default:
		return ""
	}
}

func (m MainModel) renderLoading() string {
	return m.styles.Muted.Margin(2, 4).Render("Loading instance state...")
}

// InspectAppConfig holds configuration for creating a new inspect application.
type InspectAppConfig struct {
	DeployEngine   engine.DeployEngine
	Logger         *zap.Logger
	InstanceID     string
	InstanceName   string
	Styles         *stylespkg.Styles
	Headless       bool
	HeadlessWriter io.Writer
	JSONMode       bool
}

// NewInspectApp creates a new inspect application with the given configuration.
func NewInspectApp(cfg InspectAppConfig) (*MainModel, error) {
	// Determine initial session state
	sessionState := inspectInstanceInput
	hasIdentifier := cfg.InstanceID != "" || cfg.InstanceName != ""

	if hasIdentifier || cfg.Headless {
		// Skip input form, go straight to loading
		sessionState = inspectLoading
	}

	// Create instance input form (for interactive mode without pre-set identifier)
	var instanceForm *InstanceInputFormModel
	if !hasIdentifier && !cfg.Headless {
		instanceForm = NewInstanceInputFormModel(cfg.Styles)
	}

	// Create the inspect model
	inspect := NewInspectModel(InspectModelConfig{
		DeployEngine:   cfg.DeployEngine,
		Logger:         cfg.Logger,
		InstanceID:     cfg.InstanceID,
		InstanceName:   cfg.InstanceName,
		Styles:         cfg.Styles,
		IsHeadless:     cfg.Headless,
		HeadlessWriter: cfg.HeadlessWriter,
		JSONMode:       cfg.JSONMode,
	})

	model := &MainModel{
		sessionState: sessionState,
		instanceForm: instanceForm,
		inspect:      inspect,
		instanceID:   cfg.InstanceID,
		instanceName: cfg.InstanceName,
		headless:     cfg.Headless,
		jsonMode:     cfg.JSONMode,
		engine:       cfg.DeployEngine,
		logger:       cfg.Logger,
		styles:       cfg.Styles,
	}

	return model, nil
}

// InstanceInputMsg is sent when the user enters an instance identifier.
type InstanceInputMsg struct {
	InstanceID   string
	InstanceName string
}

// InstanceInputFormModel provides a form for entering instance name/ID.
type InstanceInputFormModel struct {
	form   *huh.Form
	styles *stylespkg.Styles

	instanceName string
}

// NewInstanceInputFormModel creates a new instance input form.
func NewInstanceInputFormModel(styles *stylespkg.Styles) *InstanceInputFormModel {
	model := &InstanceInputFormModel{
		styles: styles,
	}

	model.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Key("instanceName").
				Title("Instance Name or ID").
				Description("Enter the name or ID of the instance to inspect.").
				Placeholder("my-app-production").
				Value(&model.instanceName).
				Validate(func(value string) error {
					trimmed := strings.TrimSpace(value)
					if trimmed == "" {
						return errors.New("instance name or ID is required")
					}
					return nil
				}),
		),
	).WithTheme(stylespkg.NewHuhTheme(styles.Palette))

	return model
}

// Init initializes the form.
func (m *InstanceInputFormModel) Init() tea.Cmd {
	return m.form.Init()
}

// Update handles messages.
func (m *InstanceInputFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	formModel, cmd := m.form.Update(msg)
	if form, ok := formModel.(*huh.Form); ok {
		m.form = form
	}

	if m.form.State == huh.StateCompleted {
		instanceName := strings.TrimSpace(m.form.GetString("instanceName"))
		return m, func() tea.Msg {
			return InstanceInputMsg{
				InstanceName: instanceName,
			}
		}
	}

	return m, cmd
}

// View renders the form.
func (m *InstanceInputFormModel) View() string {
	sb := strings.Builder{}
	sb.WriteString("\n")
	sb.WriteString(m.styles.Title.MarginLeft(2).Render("Inspect Instance"))
	sb.WriteString("\n\n")
	sb.WriteString(m.form.View())
	return sb.String()
}
