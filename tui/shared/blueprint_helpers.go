package shared

import (
	"path"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
)

// BlueprintSourceFromPath determines where a blueprint is loaded from, returning
// one of the consts.BlueprintSource* constants.
func BlueprintSourceFromPath(blueprintFile string) string {
	return FileLocationFromPath(blueprintFile)
}

// StripObjectStorageScheme removes the scheme prefix from an object storage path.
// For example, "s3://bucket/file.yaml" becomes "bucket/file.yaml".
// If the path doesn't have the expected scheme prefix, it's returned unchanged.
func StripObjectStorageScheme(blueprintFile, scheme string) string {
	prefix := scheme + "://"
	if after, ok := strings.CutPrefix(blueprintFile, prefix); ok {
		return after
	}
	return blueprintFile
}

// BuildObjectStorageDocumentInfo creates a BlueprintDocumentInfo for object storage sources.
// It strips the scheme prefix before extracting directory and file components,
// since path.Dir doesn't handle scheme-prefixed paths correctly.
func BuildObjectStorageDocumentInfo(blueprintFile, scheme string) types.BlueprintDocumentInfo {
	pathWithoutScheme := StripObjectStorageScheme(blueprintFile, scheme)
	return types.BlueprintDocumentInfo{
		FileSourceScheme: scheme,
		Directory:        path.Dir(pathWithoutScheme),
		BlueprintFile:    path.Base(pathWithoutScheme),
	}
}
