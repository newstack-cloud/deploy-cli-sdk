package consts

// File location constants: where the file picker reads a file from or saves it to.
const (
	FileLocationLocal     = "file"
	FileLocationS3        = "s3"
	FileLocationGCS       = "gcs"
	FileLocationAzureBlob = "azureblob"
	FileLocationHTTPS     = "https"
)

// Blueprint source aliases: where a blueprint is loaded from.
const (
	BlueprintSourceFile      = FileLocationLocal
	BlueprintSourceS3        = FileLocationS3
	BlueprintSourceGCS       = FileLocationGCS
	BlueprintSourceAzureBlob = FileLocationAzureBlob
	BlueprintSourceHTTPS     = FileLocationHTTPS
)
