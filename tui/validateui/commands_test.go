package validateui

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
	"github.com/stretchr/testify/assert"
)

// The deploy target (and other plugin config) reaches the engine via the
// validation payload's Config field; without it, transformer plugins run with
// an empty deploy target during validation. Each payload constructor must carry
// the model's operation config.
func TestCreateValidationPayload_includesOperationConfig(t *testing.T) {
	opConfig := &types.BlueprintOperationConfig{
		ContextVariables: map[string]*core.ScalarValue{
			"deployTarget": core.ScalarFromString("aws-serverless"),
		},
	}

	localModel := ValidateModel{
		blueprintFile:   "app.blueprint.yaml",
		operationConfig: opConfig,
	}
	httpsModel := ValidateModel{
		blueprintFile:   "https://example.com/dir/app.blueprint.yaml",
		operationConfig: opConfig,
	}

	localPayload, err := createValidationPayloadForLocalFile(localModel)
	assert.NoError(t, err)
	assert.Same(t, opConfig, localPayload.Config)

	objPayload, err := createValidationPayloadForObjectStorage(localModel, "s3")
	assert.NoError(t, err)
	assert.Same(t, opConfig, objPayload.Config)

	httpsPayload, err := createValidationPayloadForHTTPS(httpsModel)
	assert.NoError(t, err)
	assert.Same(t, opConfig, httpsPayload.Config)
}
