package ui

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/newstack-cloud/deploy-cli-sdk/consts"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
)

// InputLocalFileModel provides a path input for a local file that is about to be
// written, which may not exist yet.
type InputLocalFileModel struct {
	form       *huh.Form
	path       *savePathAccessor
	workingDir string
	submitted  bool
}

// huh writes the input's value from the event loop and runs the description function in
// a goroutine of its own, so the function reads the path through a lock. The binding huh
// hashes to tell when to rerun the function is only touched on the event loop.
type savePathAccessor struct {
	mu      sync.Mutex
	path    string
	binding string
}

func (a *savePathAccessor) Get() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.path
}

func (a *savePathAccessor) Set(path string) {
	a.mu.Lock()
	a.path = path
	a.mu.Unlock()
	a.binding = path
}

func (m InputLocalFileModel) Init() tea.Cmd {
	return m.form.Init()
}

func (m InputLocalFileModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	formModel, cmd := m.form.Update(msg)
	if form, ok := formModel.(*huh.Form); ok {
		m.form = form
	}

	// The form stays completed, so the selection is only sent once.
	if m.form.State == huh.StateCompleted && !m.submitted {
		m.submitted = true
		return m, tea.Batch(cmd, selectFileCmd(resolveLocalPath(m.path.Get(), m.workingDir), consts.FileLocationLocal))
	}

	return m, cmd
}

func (m InputLocalFileModel) View() string {
	return m.form.View()
}

// NewInputLocalFile creates a path input for a local file to be written.
// Relative paths are resolved against workingDir, and defaultFile pre-fills the input.
func NewInputLocalFile(
	styles *stylespkg.Styles,
	fileTypeName string,
	defaultFile string,
	workingDir string,
) *InputLocalFileModel {
	path := &savePathAccessor{path: defaultFile, binding: defaultFile}
	model := &InputLocalFileModel{
		path:       path,
		workingDir: workingDir,
	}

	model.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Key("filePath").
				Title(fmt.Sprintf("Path of the %s", fileTypeName)).
				DescriptionFunc(func() string {
					return localSaveDescription(styles, path.Get(), workingDir)
				}, &path.binding).
				Accessor(path).
				Validate(func(value string) error {
					return validateLocalSavePath(value, workingDir)
				}).
				WithWidth(80),
		),
	).WithTheme(stylespkg.NewHuhTheme(styles.Palette))

	return model
}

// The description is kept to one short line without paths: huh sizes the field when it
// is first drawn, so a description that grows and wraps pushes the input out of view.
// The overwrite warning is styled as a warning, since a change of wording alone is easy
// to miss.
func localSaveDescription(styles *stylespkg.Styles, path string, workingDir string) string {
	resolved := resolveLocalPath(path, workingDir)
	if info, err := os.Stat(resolved); err == nil && !info.IsDir() {
		return styles.Warning.Render("! This file already exists and will be overwritten.")
	}
	return "Relative paths are saved under the current directory."
}

// Export does not create directories, so a missing parent is caught here rather than
// after the state has been read.
func validateLocalSavePath(path string, workingDir string) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("file path cannot be empty")
	}

	resolved := resolveLocalPath(path, workingDir)
	if info, err := os.Stat(resolved); err == nil && info.IsDir() {
		return fmt.Errorf("%s is a directory", resolved)
	}

	parent := filepath.Dir(resolved)
	info, err := os.Stat(parent)
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("directory %s does not exist", parent)
	}
	if err != nil {
		return fmt.Errorf("directory %s cannot be used: %w", parent, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is a file, not a directory", parent)
	}
	return nil
}

func resolveLocalPath(path string, workingDir string) string {
	path = strings.TrimSpace(path)
	if rest, isHomeRelative := strings.CutPrefix(path, "~/"); isHomeRelative {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, rest)
		}
	}
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Join(workingDir, path)
}
