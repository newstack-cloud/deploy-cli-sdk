package consts

// File source constants for generalized file selection.
const (
	FileSourceLocal     = "file"
	FileSourceS3        = "s3"
	FileSourceGCS       = "gcs"
	FileSourceAzureBlob = "azureblob"
	FileSourceHTTPS     = "https"
)

// Blueprint-specific aliases for backwards compatibility.
const (
	BlueprintSourceFile      = FileSourceLocal
	BlueprintSourceS3        = FileSourceS3
	BlueprintSourceGCS       = FileSourceGCS
	BlueprintSourceAzureBlob = FileSourceAzureBlob
	BlueprintSourceHTTPS     = FileSourceHTTPS
)
