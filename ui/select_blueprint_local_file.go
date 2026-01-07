package ui

import (
	"os"

	"github.com/charmbracelet/bubbles/filepicker"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
)

func customFilePickerStyles(styles *stylespkg.Styles) filepicker.Styles {
	fpStyles := filepicker.DefaultStyles()
	fpStyles.Selected = styles.Selected
	fpStyles.File = styles.Selectable
	fpStyles.Directory = styles.Selectable
	fpStyles.Cursor = styles.Selected
	return fpStyles
}

// BlueprintLocalFilePicker creates a new filepicker model for selecting a local blueprint file
// relative to the current working directory.
func BlueprintLocalFilePicker(styles *stylespkg.Styles) (filepicker.Model, error) {
	fp := filepicker.New()
	fp.Styles = customFilePickerStyles(styles)
	fp.AllowedTypes = []string{".yaml", ".yml", ".json", ".jsonc"}

	currentDir, err := os.Getwd()
	if err != nil {
		return filepicker.Model{}, err
	}
	fp.CurrentDirectory = currentDir

	return fp, nil
}
