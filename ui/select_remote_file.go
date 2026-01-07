package ui

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/newstack-cloud/deploy-cli-sdk/consts"
	stringhelpers "github.com/newstack-cloud/deploy-cli-sdk/strings"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
)

// SelectRemoteFileConfig holds configuration for remote file selection.
type SelectRemoteFileConfig struct {
	// URLTitle is the title for the HTTPS URL input field.
	// Defaults to "HTTPS resource URL".
	URLTitle string
	// URLDescription is the description for the HTTPS URL input field.
	// Defaults to "The public URL of the file to download."
	URLDescription string
	// URLPlaceholder is the placeholder for the HTTPS URL input field.
	// Defaults to "https://assets.example.com/project.blueprint.yaml".
	URLPlaceholder string
	// BucketNameTitle overrides the bucket name title for a specific source.
	// Map keys are source constants (e.g., consts.FileSourceS3).
	BucketNameTitle map[string]string
	// BucketNameDescription overrides the bucket name description for a specific source.
	BucketNameDescription map[string]string
	// BucketNamePlaceholder overrides the bucket name placeholder for a specific source.
	BucketNamePlaceholder map[string]string
	// ObjectPathDescription overrides the object path description for a specific source.
	ObjectPathDescription map[string]string
}

// SelectRemoteFileModel provides forms for selecting files from remote sources.
type SelectRemoteFileModel struct {
	styles            *stylespkg.Styles
	selectedFile      string
	source            *string
	urlForm           *huh.Form
	objectStorageForm *huh.Form
	config            *SelectRemoteFileConfig
}

func (m SelectRemoteFileModel) Init() tea.Cmd {
	return tea.Batch(m.urlForm.Init(), m.objectStorageForm.Init())
}

func (m SelectRemoteFileModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}

	switch msg := msg.(type) {
	case SelectFileSourceMsg:
		*m.source = msg.Source
	}

	if stringhelpers.FromPointer(m.source) == consts.FileSourceHTTPS {
		urlFormModel, cmd := m.urlForm.Update(msg)
		if form, ok := urlFormModel.(*huh.Form); ok {
			m.urlForm = form
			cmds = append(cmds, cmd)
		}

		if m.urlForm.State == huh.StateCompleted {
			cmds = append(
				cmds,
				selectFileCmd(
					m.urlForm.GetString("fileUrl"),
					consts.FileSourceHTTPS,
				),
			)
		}
	} else {
		objectStorageFormModel, cmd := m.objectStorageForm.Update(msg)
		if form, ok := objectStorageFormModel.(*huh.Form); ok {
			m.objectStorageForm = form
			cmds = append(cmds, cmd)
		}

		if m.objectStorageForm.State == huh.StateCompleted {
			cmds = append(
				cmds,
				selectFileCmd(
					createRemoteObjectPath(m.objectStorageForm),
					stringhelpers.FromPointer(m.source),
				),
			)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m SelectRemoteFileModel) View() string {
	if stringhelpers.FromPointer(m.source) == consts.FileSourceHTTPS {
		return m.urlForm.View()
	}

	return m.objectStorageForm.View()
}

// NewSelectRemoteFile creates a new remote file selection model.
func NewSelectRemoteFile(
	initialFile string,
	styles *stylespkg.Styles,
	config *SelectRemoteFileConfig,
) *SelectRemoteFileModel {
	if config == nil {
		config = &SelectRemoteFileConfig{}
	}

	initSource := ""

	model := &SelectRemoteFileModel{
		styles:       styles,
		selectedFile: initialFile,
		source:       &initSource,
		config:       config,
	}

	urlTitle := config.URLTitle
	if urlTitle == "" {
		urlTitle = "HTTPS resource URL"
	}
	urlDescription := config.URLDescription
	if urlDescription == "" {
		urlDescription = "The public URL of the file to download."
	}
	urlPlaceholder := config.URLPlaceholder
	if urlPlaceholder == "" {
		urlPlaceholder = "https://assets.example.com/project.blueprint.yaml"
	}

	model.urlForm = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Key("fileUrl").
				Title(urlTitle).
				Description(urlDescription).
				Placeholder(urlPlaceholder).
				Validate(func(value string) error {
					if !isValidHTTPSURL(value) {
						return errors.New("please provide a valid HTTPS URL")
					}
					return nil
				}).
				WithWidth(80),
		),
	).WithTheme(stylespkg.NewHuhTheme(styles.Palette))

	model.objectStorageForm = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Key("bucketName").
				TitleFunc(func() string {
					return remoteBucketNameTitle(stringhelpers.FromPointer(model.source), config)
				}, model.source).
				DescriptionFunc(func() string {
					return remoteBucketNameDescription(stringhelpers.FromPointer(model.source), config)
				}, model.source).
				PlaceholderFunc(func() string {
					return remoteBucketNamePlaceholder(stringhelpers.FromPointer(model.source), config)
				}, model.source).
				Validate(func(value string) error {
					if strings.TrimSpace(value) == "" {
						return errors.New("bucket name cannot be empty")
					}
					return nil
				}).
				WithWidth(80),
			huh.NewInput().
				Key("objectPath").
				Title("Object path").
				DescriptionFunc(func() string {
					return remoteObjectPathDescription(stringhelpers.FromPointer(model.source), config)
				}, model.source).
				Placeholder("path/to/file").
				Validate(func(value string) error {
					if strings.TrimSpace(value) == "" {
						return errors.New("object path cannot be empty")
					}
					return nil
				}).
				WithWidth(80),
		),
	).WithTheme(stylespkg.NewHuhTheme(styles.Palette))

	return model
}

func remoteBucketNameTitle(source string, config *SelectRemoteFileConfig) string {
	if config != nil && config.BucketNameTitle != nil {
		if title, ok := config.BucketNameTitle[source]; ok {
			return title
		}
	}
	switch source {
	case consts.FileSourceS3:
		return "S3 bucket"
	case consts.FileSourceGCS:
		return "Google Cloud Storage bucket"
	case consts.FileSourceAzureBlob:
		return "Azure Blob Storage container"
	}
	return ""
}

func remoteBucketNameDescription(source string, config *SelectRemoteFileConfig) string {
	if config != nil && config.BucketNameDescription != nil {
		if desc, ok := config.BucketNameDescription[source]; ok {
			return desc
		}
	}
	switch source {
	case consts.FileSourceS3:
		return "The name of the S3 bucket containing the file."
	case consts.FileSourceGCS:
		return "The name of the GCS bucket containing the file."
	case consts.FileSourceAzureBlob:
		return "The name of the Azure Blob Storage container containing the file."
	}
	return ""
}

func remoteBucketNamePlaceholder(source string, config *SelectRemoteFileConfig) string {
	if config != nil && config.BucketNamePlaceholder != nil {
		if placeholder, ok := config.BucketNamePlaceholder[source]; ok {
			return placeholder
		}
	}
	switch source {
	case consts.FileSourceS3:
		return "s3-bucket"
	case consts.FileSourceGCS:
		return "gcs-bucket"
	case consts.FileSourceAzureBlob:
		return "azure-blob-container"
	}
	return ""
}

func remoteObjectPathDescription(source string, config *SelectRemoteFileConfig) string {
	if config != nil && config.ObjectPathDescription != nil {
		if desc, ok := config.ObjectPathDescription[source]; ok {
			return desc
		}
	}
	switch source {
	case consts.FileSourceS3:
		return "The path of the file in the S3 bucket."
	case consts.FileSourceGCS:
		return "The path of the file in the GCS bucket."
	case consts.FileSourceAzureBlob:
		return "The path of the file in the Azure Blob Storage container."
	}
	return ""
}

func createRemoteObjectPath(form *huh.Form) string {
	bucketName := form.GetString("bucketName")
	if !strings.HasSuffix(bucketName, "/") {
		bucketName = fmt.Sprintf("%s/", bucketName)
	}

	return fmt.Sprintf("%s%s", bucketName, form.GetString("objectPath"))
}

func isValidHTTPSURL(inputURL string) bool {
	path, err := url.Parse(inputURL)
	isValid := err == nil && path.Scheme == "https" && path.Host != ""
	return isValid
}
