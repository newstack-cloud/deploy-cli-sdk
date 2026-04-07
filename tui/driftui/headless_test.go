package driftui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/deploy-cli-sdk/headless"
	"github.com/stretchr/testify/suite"
)

type HeadlessDriftPrinterTestSuite struct {
	suite.Suite
}

func TestHeadlessDriftPrinterTestSuite(t *testing.T) {
	suite.Run(t, new(HeadlessDriftPrinterTestSuite))
}

func (s *HeadlessDriftPrinterTestSuite) newPrinter(buf *bytes.Buffer) *HeadlessDriftPrinter {
	pw := headless.NewPrefixedWriter(buf, "")
	hp := headless.NewPrinter(pw, 80)
	return NewHeadlessDriftPrinter(hp, DriftContextDeploy)
}

func (s *HeadlessDriftPrinterTestSuite) newPrinterWithContext(buf *bytes.Buffer, ctx DriftContext) *HeadlessDriftPrinter {
	pw := headless.NewPrefixedWriter(buf, "")
	hp := headless.NewPrinter(pw, 80)
	return NewHeadlessDriftPrinter(hp, ctx)
}

func (s *HeadlessDriftPrinterTestSuite) Test_PrintDriftDetected_nil_result_does_not_panic() {
	var buf bytes.Buffer
	p := s.newPrinter(&buf)
	s.NotPanics(func() {
		p.PrintDriftDetected(nil)
	})
	s.Empty(buf.String())
}

func (s *HeadlessDriftPrinterTestSuite) Test_PrintDriftDetected_nil_printer_does_not_panic() {
	p := &HeadlessDriftPrinter{printer: nil, context: DriftContextDeploy}
	result := &container.ReconciliationCheckResult{}
	s.NotPanics(func() {
		p.PrintDriftDetected(result)
	})
}

func (s *HeadlessDriftPrinterTestSuite) Test_PrintDriftDetected_resources_only() {
	var buf bytes.Buffer
	p := s.newPrinter(&buf)

	result := &container.ReconciliationCheckResult{
		Resources: []container.ResourceReconcileResult{
			{
				ResourceName: "myBucket",
				ResourceType: "aws/s3/bucket",
				Type:         container.ReconciliationTypeDrift,
			},
		},
	}
	p.PrintDriftDetected(result)

	output := buf.String()
	s.Contains(output, "Drift detected")
	s.Contains(output, "Resources with drift:")
	s.Contains(output, "myBucket")
	s.Contains(output, "aws/s3/bucket")
	s.NotContains(output, "Links with drift:")
}

func (s *HeadlessDriftPrinterTestSuite) Test_PrintDriftDetected_links_only() {
	var buf bytes.Buffer
	p := s.newPrinter(&buf)

	result := &container.ReconciliationCheckResult{
		Links: []container.LinkReconcileResult{
			{
				LinkName: "myLink",
				Type:     container.ReconciliationTypeDrift,
			},
		},
	}
	p.PrintDriftDetected(result)

	output := buf.String()
	s.Contains(output, "Drift detected")
	s.Contains(output, "Links with drift:")
	s.Contains(output, "myLink")
	s.NotContains(output, "Resources with drift:")
}

func (s *HeadlessDriftPrinterTestSuite) Test_PrintDriftDetected_resources_and_links() {
	var buf bytes.Buffer
	p := s.newPrinter(&buf)

	result := &container.ReconciliationCheckResult{
		Resources: []container.ResourceReconcileResult{
			{
				ResourceName: "myBucket",
				ResourceType: "aws/s3/bucket",
				Type:         container.ReconciliationTypeDrift,
			},
		},
		Links: []container.LinkReconcileResult{
			{
				LinkName: "myLink",
				Type:     container.ReconciliationTypeDrift,
			},
		},
	}
	p.PrintDriftDetected(result)

	output := buf.String()
	s.Contains(output, "Resources with drift:")
	s.Contains(output, "myBucket")
	s.Contains(output, "Links with drift:")
	s.Contains(output, "myLink")
}

func (s *HeadlessDriftPrinterTestSuite) Test_PrintDriftDetected_resource_modified_fields() {
	var buf bytes.Buffer
	p := s.newPrinter(&buf)

	oldVal := "old-value"
	newVal := "new-value"
	result := &container.ReconciliationCheckResult{
		Resources: []container.ResourceReconcileResult{
			{
				ResourceName: "myBucket",
				ResourceType: "aws/s3/bucket",
				Type:         container.ReconciliationTypeDrift,
				Changes: &provider.Changes{
					ModifiedFields: []provider.FieldChange{
						{
							FieldPath: "spec.tags",
							PrevValue: &core.MappingNode{
								Scalar: &core.ScalarValue{StringValue: &oldVal},
							},
							NewValue: &core.MappingNode{
								Scalar: &core.ScalarValue{StringValue: &newVal},
							},
						},
					},
				},
			},
		},
	}
	p.PrintDriftDetected(result)

	output := buf.String()
	s.Contains(output, "spec.tags")
	s.Contains(output, "old-value")
	s.Contains(output, "new-value")
}

func (s *HeadlessDriftPrinterTestSuite) Test_PrintDriftDetected_resource_new_fields() {
	var buf bytes.Buffer
	p := s.newPrinter(&buf)

	newVal := "added-value"
	result := &container.ReconciliationCheckResult{
		Resources: []container.ResourceReconcileResult{
			{
				ResourceName: "myBucket",
				ResourceType: "aws/s3/bucket",
				Type:         container.ReconciliationTypeDrift,
				Changes: &provider.Changes{
					NewFields: []provider.FieldChange{
						{
							FieldPath: "spec.newTag",
							NewValue: &core.MappingNode{
								Scalar: &core.ScalarValue{StringValue: &newVal},
							},
						},
					},
				},
			},
		},
	}
	p.PrintDriftDetected(result)

	output := buf.String()
	s.Contains(output, "spec.newTag")
	s.Contains(output, "added-value")
	// New fields use "+" prefix
	s.True(strings.Contains(output, "+ spec.newTag"))
}

func (s *HeadlessDriftPrinterTestSuite) Test_PrintDriftDetected_resource_removed_fields() {
	var buf bytes.Buffer
	p := s.newPrinter(&buf)

	result := &container.ReconciliationCheckResult{
		Resources: []container.ResourceReconcileResult{
			{
				ResourceName: "myBucket",
				ResourceType: "aws/s3/bucket",
				Type:         container.ReconciliationTypeDrift,
				Changes: &provider.Changes{
					RemovedFields: []string{"spec.oldTag"},
				},
			},
		},
	}
	p.PrintDriftDetected(result)

	output := buf.String()
	s.Contains(output, "spec.oldTag")
	// Removed fields use "-" prefix
	s.True(strings.Contains(output, "- spec.oldTag"))
}

func (s *HeadlessDriftPrinterTestSuite) Test_PrintDriftDetected_interrupted_resource() {
	var buf bytes.Buffer
	p := s.newPrinter(&buf)

	result := &container.ReconciliationCheckResult{
		Resources: []container.ResourceReconcileResult{
			{
				ResourceName:   "interruptedRes",
				ResourceType:   "aws/ec2/instance",
				Type:           container.ReconciliationTypeInterrupted,
				ResourceExists: false,
			},
		},
	}
	p.PrintDriftDetected(result)

	output := buf.String()
	s.Contains(output, "interruptedRes")
	s.Contains(output, "INTERRUPTED")
	s.Contains(output, "Resource exists: No")
}

func (s *HeadlessDriftPrinterTestSuite) Test_PrintDriftDetected_interrupted_resource_exists() {
	var buf bytes.Buffer
	p := s.newPrinter(&buf)

	result := &container.ReconciliationCheckResult{
		Resources: []container.ResourceReconcileResult{
			{
				ResourceName:   "interruptedRes",
				ResourceType:   "aws/ec2/instance",
				Type:           container.ReconciliationTypeInterrupted,
				ResourceExists: true,
			},
		},
	}
	p.PrintDriftDetected(result)

	output := buf.String()
	s.Contains(output, "Resource exists: Yes")
}

func (s *HeadlessDriftPrinterTestSuite) Test_PrintDriftDetected_link_data_updates() {
	var buf bytes.Buffer
	p := s.newPrinter(&buf)

	result := &container.ReconciliationCheckResult{
		Links: []container.LinkReconcileResult{
			{
				LinkName: "myLink",
				Type:     container.ReconciliationTypeDrift,
				LinkDataUpdates: map[string]*core.MappingNode{
					"connection.endpoint": nil,
				},
			},
		},
	}
	p.PrintDriftDetected(result)

	output := buf.String()
	s.Contains(output, "myLink")
	s.Contains(output, "connection.endpoint")
	s.Contains(output, "link data affected")
}

func (s *HeadlessDriftPrinterTestSuite) Test_PrintDriftDetected_child_path_resources() {
	var buf bytes.Buffer
	p := s.newPrinter(&buf)

	result := &container.ReconciliationCheckResult{
		Resources: []container.ResourceReconcileResult{
			{
				ResourceName: "childResource",
				ResourceType: "aws/lambda/function",
				ChildPath:    "childBlueprint",
				Type:         container.ReconciliationTypeDrift,
			},
		},
	}
	p.PrintDriftDetected(result)

	output := buf.String()
	s.Contains(output, "Child blueprints with drift:")
	s.Contains(output, "childBlueprint")
	s.Contains(output, "childResource")
	// Parent-level resources section should not appear
	s.NotContains(output, "Resources with drift:")
}

func (s *HeadlessDriftPrinterTestSuite) Test_PrintDriftDetected_nested_child_path() {
	var buf bytes.Buffer
	p := s.newPrinter(&buf)

	result := &container.ReconciliationCheckResult{
		Resources: []container.ResourceReconcileResult{
			{
				ResourceName: "deepResource",
				ResourceType: "aws/dynamodb/table",
				ChildPath:    "child1.child2",
				Type:         container.ReconciliationTypeDrift,
			},
		},
	}
	p.PrintDriftDetected(result)

	output := buf.String()
	s.Contains(output, "child1")
	s.Contains(output, "child2")
	s.Contains(output, "deepResource")
}

func (s *HeadlessDriftPrinterTestSuite) Test_PrintDriftDetected_child_path_links() {
	var buf bytes.Buffer
	p := s.newPrinter(&buf)

	result := &container.ReconciliationCheckResult{
		Links: []container.LinkReconcileResult{
			{
				LinkName:  "childLink",
				ChildPath: "childBlueprint",
				Type:      container.ReconciliationTypeDrift,
			},
		},
	}
	p.PrintDriftDetected(result)

	output := buf.String()
	s.Contains(output, "Child blueprints with drift:")
	s.Contains(output, "childBlueprint")
	s.Contains(output, "childLink")
}

func (s *HeadlessDriftPrinterTestSuite) Test_PrintDriftDetected_hint_deploy_context() {
	var buf bytes.Buffer
	p := s.newPrinterWithContext(&buf, DriftContextDeploy)

	result := &container.ReconciliationCheckResult{}
	p.PrintDriftDetected(result)

	output := buf.String()
	s.Contains(output, "--force")
}

func (s *HeadlessDriftPrinterTestSuite) Test_PrintDriftDetected_hint_stage_context() {
	var buf bytes.Buffer
	p := s.newPrinterWithContext(&buf, DriftContextStage)

	result := &container.ReconciliationCheckResult{}
	p.PrintDriftDetected(result)

	output := buf.String()
	s.Contains(output, "--skip-drift-check")
}

func (s *HeadlessDriftPrinterTestSuite) Test_PrintDriftDetected_interrupted_with_external_state_no_changes() {
	var buf bytes.Buffer
	p := s.newPrinter(&buf)

	stateVal := "running"
	result := &container.ReconciliationCheckResult{
		Resources: []container.ResourceReconcileResult{
			{
				ResourceName:   "interruptedRes",
				ResourceType:   "aws/ec2/instance",
				Type:           container.ReconciliationTypeInterrupted,
				ResourceExists: true,
				ExternalState: &core.MappingNode{
					Scalar: &core.ScalarValue{StringValue: &stateVal},
				},
			},
		},
	}
	p.PrintDriftDetected(result)

	output := buf.String()
	s.Contains(output, "External state available")
}

func (s *HeadlessDriftPrinterTestSuite) Test_PrintDriftDetected_child_node_shows_drifted_count() {
	var buf bytes.Buffer
	p := s.newPrinter(&buf)

	result := &container.ReconciliationCheckResult{
		Resources: []container.ResourceReconcileResult{
			{
				ResourceName: "res1",
				ResourceType: "aws/s3/bucket",
				ChildPath:    "childA",
				Type:         container.ReconciliationTypeDrift,
			},
			{
				ResourceName: "res2",
				ResourceType: "aws/s3/bucket",
				ChildPath:    "childA",
				Type:         container.ReconciliationTypeDrift,
			},
		},
	}
	p.PrintDriftDetected(result)

	output := buf.String()
	// Should show "2 drifted" in the child node header
	s.Contains(output, "2 drifted")
}

func (s *HeadlessDriftPrinterTestSuite) Test_NewHeadlessDriftPrinter_stores_context() {
	var buf bytes.Buffer
	pw := headless.NewPrefixedWriter(&buf, "")
	hp := headless.NewPrinter(pw, 80)
	p := NewHeadlessDriftPrinter(hp, DriftContextDestroy)
	s.Equal(DriftContextDestroy, p.context)
}
