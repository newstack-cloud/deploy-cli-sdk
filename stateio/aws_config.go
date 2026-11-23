package stateio

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

// Static credentials are used only when both parts are set, as the deploy engine does, and
// the AWS environment's credentials chain otherwise. An empty region leaves the region to
// the environment too.
func loadAWSConfig(ctx context.Context, region string, accessKeyID string, secretAccessKey string) (aws.Config, error) {
	opts := []func(*awsconfig.LoadOptions) error{}
	if region != "" {
		opts = append(opts, awsconfig.WithRegion(region))
	}
	if accessKeyID != "" && secretAccessKey != "" {
		opts = append(opts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, ""),
		))
	}
	return awsconfig.LoadDefaultConfig(ctx, opts...)
}
