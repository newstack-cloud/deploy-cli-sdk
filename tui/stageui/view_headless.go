package stageui

import (
	"fmt"
	"sort"

	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	engineerrors "github.com/newstack-cloud/bluelink/libs/deploy-engine-client/errors"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
	"github.com/newstack-cloud/deploy-cli-sdk/headless"
	sdkstrings "github.com/newstack-cloud/deploy-cli-sdk/strings"
	"github.com/newstack-cloud/deploy-cli-sdk/tui/driftui"
	"github.com/newstack-cloud/deploy-cli-sdk/tui/outpututil"
	"github.com/newstack-cloud/deploy-cli-sdk/tui/shared"
)

func (m *StageModel) printHeadlessHeader() {
	w := m.printer.Writer()
	w.Println("Starting change staging...")
	w.Printf("Changeset: %s\n", m.changesetID)
	w.DoubleSeparator(72)
	w.PrintlnEmpty()
}

func (m *StageModel) printHeadlessResourceEvent(data *types.ResourceChangesEventData) {
	action := m.determineResourceAction(data)
	suffix := ""
	if data.New {
		suffix = "(new)"
	}
	m.printer.ProgressItem("✓", "resource", data.ResourceName, string(action), suffix)
}

func (m *StageModel) printHeadlessChildEvent(data *types.ChildChangesEventData) {
	action := m.determineChildAction(data)
	resourceCount := len(data.Changes.NewResources) + len(data.Changes.ResourceChanges)
	suffix := ""
	if data.New {
		suffix = fmt.Sprintf("(new, %d %s)", resourceCount, sdkstrings.Pluralize(resourceCount, "resource", "resources"))
	} else {
		suffix = fmt.Sprintf("(%d %s)", resourceCount, sdkstrings.Pluralize(resourceCount, "resource", "resources"))
	}
	m.printer.ProgressItem("✓", "child", data.ChildBlueprintName, string(action), suffix)
}

func (m *StageModel) printHeadlessLinkEvent(data *types.LinkChangesEventData) {
	action := m.determineLinkAction(data)
	linkName := fmt.Sprintf("%s::%s", data.ResourceAName, data.ResourceBName)
	suffix := ""
	if data.New {
		suffix = "(new)"
	}
	m.printer.ProgressItem("✓", "link", linkName, string(action), suffix)
}

func (m *StageModel) printHeadlessSummary() {
	w := m.printer.Writer()
	w.PrintlnEmpty()
	w.DoubleSeparator(72)
	w.Println("Change staging complete")
	w.DoubleSeparator(72)
	w.PrintlnEmpty()

	m.printHeadlessGroupedItems()

	resources, children, links := m.countByType()
	create, update, delete, recreate, retain := m.countChangeSummary()
	hasDeploymentChanges := create > 0 || update > 0 || delete > 0 || recreate > 0 || retain > 0

	if m.completeChanges != nil {
		showExports := HasAnyExportChanges(m.completeChanges) ||
			(hasDeploymentChanges && HasAnyExportsToShow(m.completeChanges))
		if showExports {
			m.printHeadlessExportChanges(m.completeChanges, "")
		}
	}

	w.DoubleSeparator(72)
	w.Printf("Complete: %d %s, %d %s, %d %s\n",
		resources, sdkstrings.Pluralize(resources, "resource", "resources"),
		children, sdkstrings.Pluralize(children, "child", "children"),
		links, sdkstrings.Pluralize(links, "link", "links"))
	w.Printf("Actions: %d create, %d update, %d delete, %d recreate, %d retain\n", create, update, delete, recreate, retain)
	w.PrintlnEmpty()
	w.Printf("Changeset ID: %s\n", m.changesetID)
	w.PrintlnEmpty()

	if !hasDeploymentChanges {
		w.Println("No changes to apply.")
		return
	}

	m.printHeadlessApplyHint()
}

func (m *StageModel) printHeadlessApplyHint() {
	var cmd string
	if m.destroy {
		cmd = fmt.Sprintf("bluelink destroy --changeset-id %s", m.changesetID)
	} else {
		cmd = fmt.Sprintf("bluelink deploy --changeset-id %s", m.changesetID)
	}
	if m.instanceName != "" {
		cmd += fmt.Sprintf(" --instance-name %s", m.instanceName)
	} else if m.instanceID != "" {
		cmd += fmt.Sprintf(" --instance-id %s", m.instanceID)
	} else {
		cmd += " --instance-name <name>"
	}
	m.printer.NextStep("To apply these changes, run:", cmd)
}

func (m *StageModel) printHeadlessGroupedItems() {
	var resources, others []StageItem
	for _, item := range m.items {
		if item.Type == ItemTypeResource && item.ParentChild == "" {
			resources = append(resources, item)
		} else {
			others = append(others, item)
		}
	}

	infos := make([]shared.HeadlessResourceInfo, 0, len(resources))
	resourceMap := make(map[string]StageItem, len(resources))
	for _, item := range resources {
		meta := resolveStageResourceMetadata(&item)
		infos = append(infos, shared.HeadlessResourceInfo{
			Path: item.Name, Name: item.Name, Metadata: meta,
		})
		resourceMap[item.Name] = item
	}

	groups, ungrouped := shared.GroupHeadlessResources(infos)
	w := m.printer.Writer()

	for _, group := range groups {
		w.PrintlnEmpty()
		w.Printf("[%s] %s\n", group.Group.GroupType, group.Group.GroupName)
		for _, res := range group.Resources {
			m.printHeadlessItemDetails(resourceMap[res.Name])
		}
	}
	for _, res := range ungrouped {
		m.printHeadlessItemDetails(resourceMap[res.Name])
	}
	for _, item := range others {
		m.printHeadlessItemDetails(item)
	}
}

func resolveStageResourceMetadata(item *StageItem) *state.ResourceMetadataState {
	if c, ok := item.Changes.(*provider.Changes); ok && c != nil {
		if rs := c.AppliedResourceInfo.CurrentResourceState; rs != nil && rs.Metadata != nil {
			return rs.Metadata
		}
	}
	if item.ResourceState != nil && item.ResourceState.Metadata != nil {
		return item.ResourceState.Metadata
	}
	return nil
}

func (m *StageModel) printHeadlessItemDetails(item StageItem) {
	w := m.printer.Writer()

	displayName := item.Name
	if item.ParentChild != "" {
		displayName = fmt.Sprintf("%s.%s", item.ParentChild, item.Name)
	}

	m.printer.ItemHeader(string(item.Type), displayName, string(item.Action))
	w.SingleSeparator(72)

	switch item.Type {
	case ItemTypeResource:
		m.printHeadlessResourceItemDetails(&item)
	case ItemTypeChild:
		m.printHeadlessChildItemDetails(&item)
	case ItemTypeLink:
		m.printHeadlessLinkItemDetails(&item)
	}

	w.PrintlnEmpty()
	w.PrintlnEmpty()
}

func (m *StageModel) printHeadlessResourceItemDetails(item *StageItem) {
	if item.Removed && item.ResourceState != nil {
		m.printHeadlessResourceCurrentState(item.ResourceState)
		return
	}

	if resourceChanges, ok := item.Changes.(*provider.Changes); ok {
		m.printHeadlessResourceChanges(resourceChanges)
	}
}

func (m *StageModel) printHeadlessResourceCurrentState(resourceState *state.ResourceState) {
	w := m.printer.Writer()

	w.Printf("Resource ID: %s\n", resourceState.ResourceID)

	if resourceState.Type != "" {
		w.Printf("Type: %s\n", resourceState.Type)
	}

	if resourceState.SpecData == nil {
		return
	}

	fields := outpututil.CollectOutputFields(resourceState.SpecData, resourceState.ComputedFields)
	if len(fields) == 0 {
		return
	}

	sort.Slice(fields, func(i, j int) bool {
		return fields[i].Name < fields[j].Name
	})
	w.PrintlnEmpty()
	w.Println("Current Outputs:")
	for _, field := range fields {
		w.Printf("  %s: %s\n", field.Name, field.Value)
	}
}

func (m *StageModel) printHeadlessChildItemDetails(item *StageItem) {
	if item.Removed {
		m.printer.Writer().Println("Child blueprint will be destroyed")
		return
	}

	if childChanges, ok := item.Changes.(*changes.BlueprintChanges); ok {
		m.printHeadlessChildChanges(childChanges)
	}
}

func (m *StageModel) printHeadlessLinkItemDetails(item *StageItem) {
	w := m.printer.Writer()

	if item.Removed && item.LinkState != nil {
		w.Printf("Link ID: %s\n", item.LinkState.LinkID)
		return
	}

	if linkChanges, ok := item.Changes.(*provider.LinkChanges); ok {
		m.printHeadlessLinkChanges(linkChanges)
	}
}

func (m *StageModel) printHeadlessResourceChanges(resourceChanges *provider.Changes) {
	w := m.printer.Writer()

	hasFieldChanges := provider.ChangesHasFieldChanges(resourceChanges)
	hasOutboundLinkChanges := len(resourceChanges.NewOutboundLinks) > 0 ||
		len(resourceChanges.OutboundLinkChanges) > 0 ||
		len(resourceChanges.RemovedOutboundLinks) > 0

	if !hasFieldChanges && !hasOutboundLinkChanges {
		m.printer.NoChanges()
		return
	}

	w.Println("Field Changes:")
	if hasFieldChanges {
		for _, field := range resourceChanges.NewFields {
			m.printer.FieldAdd(field.FieldPath, headless.FormatMappingNode(field.NewValue))
		}

		for _, field := range resourceChanges.ModifiedFields {
			m.printer.FieldModify(
				field.FieldPath,
				headless.FormatMappingNode(field.PrevValue),
				headless.FormatMappingNode(field.NewValue),
			)
		}

		for _, fieldPath := range resourceChanges.RemovedFields {
			m.printer.FieldRemove(fieldPath)
		}
	} else {
		w.Println("  None")
	}

	if hasOutboundLinkChanges {
		w.PrintlnEmpty()
		m.printHeadlessOutboundLinkChanges(resourceChanges)
	}
}

func (m *StageModel) printHeadlessOutboundLinkChanges(resourceChanges *provider.Changes) {
	w := m.printer.Writer()
	w.Println("Outbound Link Changes:")

	for linkName, linkChanges := range resourceChanges.NewOutboundLinks {
		w.Printf("  + %s (new link)\n", linkName)
		m.printHeadlessLinkFieldChanges(&linkChanges, "      ")
	}

	for linkName, linkChanges := range resourceChanges.OutboundLinkChanges {
		w.Printf("  ± %s (link updated)\n", linkName)
		m.printHeadlessLinkFieldChanges(&linkChanges, "      ")
	}

	for _, linkName := range resourceChanges.RemovedOutboundLinks {
		w.Printf("  - %s (link removed)\n", linkName)
	}
}

func (m *StageModel) printHeadlessLinkFieldChanges(linkChanges *provider.LinkChanges, indent string) {
	if linkChanges == nil {
		return
	}

	hasChanges := len(linkChanges.NewFields) > 0 ||
		len(linkChanges.ModifiedFields) > 0 ||
		len(linkChanges.RemovedFields) > 0

	if !hasChanges {
		return
	}

	w := m.printer.Writer()

	for _, field := range linkChanges.NewFields {
		w.Printf("%s+ %s: %s\n", indent, field.FieldPath, headless.FormatMappingNode(field.NewValue))
	}

	for _, field := range linkChanges.ModifiedFields {
		w.Printf("%s± %s: %s -> %s\n", indent, field.FieldPath,
			headless.FormatMappingNode(field.PrevValue),
			headless.FormatMappingNode(field.NewValue))
	}

	for _, fieldPath := range linkChanges.RemovedFields {
		w.Printf("%s- %s\n", indent, fieldPath)
	}
}

func (m *StageModel) printHeadlessChildChanges(childChanges *changes.BlueprintChanges) {
	newCount := len(childChanges.NewResources)
	updateCount := len(childChanges.ResourceChanges)
	removeCount := len(childChanges.RemovedResources)

	m.printer.CountSummary(newCount, "resource", "resources", "to be created")
	m.printer.CountSummary(updateCount, "resource", "resources", "to be updated")
	m.printer.CountSummary(removeCount, "resource", "resources", "to be removed")

	if newCount == 0 && updateCount == 0 && removeCount == 0 {
		m.printer.NoChanges()
	}
}

func (m *StageModel) printHeadlessLinkChanges(linkChanges *provider.LinkChanges) {
	hasChanges := len(linkChanges.NewFields) > 0 ||
		len(linkChanges.ModifiedFields) > 0 ||
		len(linkChanges.RemovedFields) > 0

	if !hasChanges {
		m.printer.NoChanges()
		return
	}

	for _, field := range linkChanges.NewFields {
		m.printer.FieldAdd(field.FieldPath, headless.FormatMappingNode(field.NewValue))
	}

	for _, field := range linkChanges.ModifiedFields {
		m.printer.FieldModify(
			field.FieldPath,
			headless.FormatMappingNode(field.PrevValue),
			headless.FormatMappingNode(field.NewValue),
		)
	}

	for _, fieldPath := range linkChanges.RemovedFields {
		m.printer.FieldRemove(fieldPath)
	}
}

func (m *StageModel) printHeadlessError(err error) {
	w := m.printer.Writer()
	w.PrintlnEmpty()

	if clientErr, isValidation := engineerrors.IsValidationError(err); isValidation {
		m.printHeadlessValidationError(clientErr)
		return
	}

	if streamErr, ok := err.(*engineerrors.StreamError); ok {
		m.printHeadlessStreamError(streamErr)
		return
	}

	w.Println("✗ Error during change staging")
	w.PrintlnEmpty()
	w.Printf("  Error: %s\n", err.Error())
}

func (m *StageModel) printHeadlessValidationError(clientErr *engineerrors.ClientError) {
	w := m.printer.Writer()
	w.Println("✗ Failed to create changeset")
	w.PrintlnEmpty()
	w.Println("The following issues must be resolved in the blueprint before changes can be staged:")
	w.PrintlnEmpty()

	if len(clientErr.ValidationErrors) > 0 {
		w.Println("Validation Errors:")
		w.SingleSeparator(72)
		for _, valErr := range clientErr.ValidationErrors {
			location := valErr.Location
			if location == "" {
				location = "unknown"
			}
			w.Printf("  • %s: %s\n", location, valErr.Message)
		}
		w.PrintlnEmpty()
	}

	if len(clientErr.ValidationDiagnostics) > 0 {
		w.Println("Blueprint Diagnostics:")
		w.SingleSeparator(72)
		for _, diag := range clientErr.ValidationDiagnostics {
			m.printHeadlessDiagnostic(diag)
		}
		w.PrintlnEmpty()
	}

	if len(clientErr.ValidationErrors) == 0 && len(clientErr.ValidationDiagnostics) == 0 {
		w.Printf("  %s\n", clientErr.Message)
	}
}

func (m *StageModel) printHeadlessStreamError(streamErr *engineerrors.StreamError) {
	w := m.printer.Writer()
	w.Println("✗ Error during change staging")
	w.PrintlnEmpty()
	w.Println("The following issues occurred during change staging:")
	w.PrintlnEmpty()
	w.Printf("  %s\n", streamErr.Event.Message)
	w.PrintlnEmpty()

	if len(streamErr.Event.Diagnostics) > 0 {
		w.Println("Diagnostics:")
		w.SingleSeparator(72)
		for _, diag := range streamErr.Event.Diagnostics {
			m.printHeadlessDiagnostic(diag)
		}
		w.PrintlnEmpty()
	}
}

func (m *StageModel) printHeadlessDiagnostic(diag *core.Diagnostic) {
	level := headless.DiagnosticLevelFromCore(diag.Level)
	levelName := headless.DiagnosticLevelName(level)

	line, col := 0, 0
	if diag.Range != nil {
		line = diag.Range.Start.Line
		col = diag.Range.Start.Column
	}

	m.printer.Diagnostic(levelName, diag.Message, line, col)
}

func (m *StageModel) printHeadlessDriftDetected() {
	printer := driftui.NewHeadlessDriftPrinter(m.printer, driftui.DriftContextStage)
	printer.PrintDriftDetected(m.driftResult)
}

func (m *StageModel) printHeadlessExportChanges(bc *changes.BlueprintChanges, prefix string) {
	if bc == nil {
		return
	}

	newCount, modifiedCount, removedCount, _ := countExportChanges(bc)

	if newCount > 0 || modifiedCount > 0 || removedCount > 0 {
		m.printHeadlessExportChangesSection(bc, prefix, newCount, modifiedCount, removedCount)
	}

	m.printHeadlessChildExportChanges(bc, prefix)
}

func (m *StageModel) printHeadlessExportChangesSection(bc *changes.BlueprintChanges, prefix string, newCount, modifiedCount, removedCount int) {
	w := m.printer.Writer()

	if prefix == "" {
		m.printer.ItemHeader("exports", "(root)", "")
	} else {
		m.printer.ItemHeader("exports", prefix, "")
	}
	w.SingleSeparator(72)

	if newCount > 0 {
		m.printHeadlessNewExports(bc)
	}

	if modifiedCount > 0 {
		m.printHeadlessModifiedExports(bc)
	}

	if removedCount > 0 {
		w.Println("Removed Exports:")
		for _, name := range bc.RemovedExports {
			m.printer.FieldRemove(name)
		}
	}

	w.PrintlnEmpty()
}

func (m *StageModel) printHeadlessNewExports(bc *changes.BlueprintChanges) {
	m.printer.Writer().Println("New Exports:")
	for name, change := range bc.NewExports {
		m.printHeadlessExportField(name, &change, bc.ResolveOnDeploy, false)
	}
	for name, change := range bc.ExportChanges {
		if change.PrevValue == nil {
			m.printHeadlessExportField(name, &change, bc.ResolveOnDeploy, false)
		}
	}
}

func (m *StageModel) printHeadlessModifiedExports(bc *changes.BlueprintChanges) {
	m.printer.Writer().Println("Modified Exports:")
	for name, change := range bc.ExportChanges {
		if change.PrevValue != nil {
			m.printHeadlessExportField(name, &change, bc.ResolveOnDeploy, true)
		}
	}
}

func (m *StageModel) printHeadlessChildExportChanges(bc *changes.BlueprintChanges, prefix string) {
	for childName, newChild := range bc.NewChildren {
		childPrefix := joinExportPath(prefix, childName)
		childChanges := &changes.BlueprintChanges{
			NewExports:      newChild.NewExports,
			NewChildren:     newChild.NewChildren,
			ResolveOnDeploy: newChild.ResolveOnDeploy,
		}
		m.printHeadlessExportChanges(childChanges, childPrefix)
	}

	for childName, childChanges := range bc.ChildChanges {
		childPrefix := joinExportPath(prefix, childName)
		m.printHeadlessExportChanges(&childChanges, childPrefix)
	}
}

func (m *StageModel) printHeadlessExportField(
	name string,
	change *provider.FieldChange,
	resolveOnDeploy []string,
	isModified bool,
) {
	isComputedAtDeploy := isExportComputedAtDeploy(name, resolveOnDeploy)

	if isModified {
		prevValue := "(none)"
		if change.PrevValue != nil {
			prevValue = headless.FormatMappingNode(change.PrevValue)
		}
		newValue := "(known on deploy)"
		if !isComputedAtDeploy && change.NewValue != nil {
			newValue = headless.FormatMappingNode(change.NewValue)
		}
		m.printer.FieldModify(name, prevValue, newValue)
	} else {
		value := "(known on deploy)"
		if !isComputedAtDeploy && change.NewValue != nil {
			value = headless.FormatMappingNode(change.NewValue)
		}
		m.printer.FieldAdd(name, value)
	}
}
