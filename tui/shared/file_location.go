package shared

import (
	"strings"

	"github.com/newstack-cloud/deploy-cli-sdk/consts"
)

// FileLocationFromPath returns the consts.FileLocation* constant for a path's URL scheme
// (https://, s3://, gcs:// or azureblob://), or consts.FileLocationLocal when it has none.
func FileLocationFromPath(filePath string) string {
	if strings.HasPrefix(filePath, "https://") {
		return consts.FileLocationHTTPS
	}
	if strings.HasPrefix(filePath, "s3://") {
		return consts.FileLocationS3
	}
	if strings.HasPrefix(filePath, "gcs://") {
		return consts.FileLocationGCS
	}
	if strings.HasPrefix(filePath, "azureblob://") {
		return consts.FileLocationAzureBlob
	}
	return consts.FileLocationLocal
}
