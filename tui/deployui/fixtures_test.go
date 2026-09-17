package deployui

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/deploy-cli-sdk/tui/shared"
)

// A Celerity-shaped deployment, which is one API expanded into four concrete resources,
// two handlers each expanded into a lambda and a role, and standalone infra.
type expanded struct {
	name         string
	resourceType string
	group        string
	groupType    string
}

func celerityResources() []expanded {
	return []expanded{
		{"ordersApi_http_api", "aws/apigatewayv2/api", "ordersApi", "celerity/api"},
		{"ordersApi_http_stage", "aws/apigatewayv2/stage", "ordersApi", "celerity/api"},
		{"ordersApi_domain", "aws/apigatewayv2/domainName", "ordersApi", "celerity/api"},
		{"ordersApi_http_api_mapping_0", "aws/apigatewayv2/apiMapping", "ordersApi", "celerity/api"},
		{"createOrder_lambda", "aws/lambda/function", "createOrder", "celerity/handler"},
		{"createOrder_role", "aws/iam/role", "createOrder", "celerity/handler"},
		{"listOrders_lambda", "aws/lambda/function", "listOrders", "celerity/handler"},
		{"listOrders_role", "aws/iam/role", "listOrders", "celerity/handler"},
		{"tasksDatastore_table", "aws/dynamodb/table", "tasksDatastore", "celerity/datastore"},
		{"filesBucket_bucket", "aws/s3/bucket", "filesBucket", "celerity/bucket"},
		{"appVpc_vpc", "aws/ec2/vpc", "appVpc", "celerity/vpc"},
		{"appVpc_subnet", "aws/ec2/subnet", "appVpc", "celerity/vpc"},
	}
}

func celerityLinks() []string {
	return []string{
		// internal to the ordersApi group
		"ordersApi_http_api::ordersApi_http_stage",
		"ordersApi_domain::ordersApi_http_api_mapping_0",
		// internal to a handler group
		"createOrder_lambda::createOrder_role",
		"listOrders_lambda::listOrders_role",
		// the VPC links out to everything, as ambient plumbing
		"appVpc_vpc::createOrder_lambda",
		"appVpc_vpc::listOrders_lambda",
		"appVpc_vpc::tasksDatastore_table",
		// across groups
		"ordersApi_http_api::createOrder_lambda",
		"ordersApi_http_api::listOrders_lambda",
		"createOrder_lambda::tasksDatastore_table",
		"listOrders_lambda::filesBucket_bucket",
	}
}

func annotationsFor(e expanded) *core.MappingNode {
	annotations := map[string]string{
		shared.AnnotationSourceAbstractName: e.group,
		shared.AnnotationSourceAbstractType: e.groupType,
		shared.AnnotationResourceCategory:   shared.ResourceCategoryInfrastructure,
	}
	return core.MappingNodeFromStringMap(annotations)
}

// Mirrors what the engine persists for a first deployment where
// every resource is new, and links hang off their source resource.
func harnessChangeset() *changes.BlueprintChanges {
	newResources := map[string]provider.Changes{}
	for _, e := range celerityResources() {
		newResources[e.name] = provider.Changes{
			AppliedResourceInfo: provider.ResourceInfo{
				ResourceName: e.name,
				ResourceWithResolvedSubs: &provider.ResolvedResource{
					Type: &schema.ResourceTypeWrapper{Value: e.resourceType},
					Metadata: &provider.ResolvedResourceMetadata{
						Annotations: annotationsFor(e),
					},
				},
			},
			NewOutboundLinks: map[string]provider.LinkChanges{},
		}
	}
	for _, link := range celerityLinks() {
		a := ExtractResourceAFromLinkName(link)
		b := ExtractResourceBFromLinkName(link)
		if rc, ok := newResources[a]; ok {
			rc.NewOutboundLinks[b] = provider.LinkChanges{}
			newResources[a] = rc
		}
	}
	return &changes.BlueprintChanges{NewResources: newResources}
}

func harnessInstanceState() *state.InstanceState {
	resources := map[string]*state.ResourceState{}
	resourceIDs := map[string]string{}
	for _, e := range celerityResources() {
		id := "res-" + e.name
		resourceIDs[e.name] = id
		resources[id] = &state.ResourceState{
			ResourceID: id,
			Name:       e.name,
			Type:       e.resourceType,
			Status:     core.ResourceStatusCreated,
			Metadata: &state.ResourceMetadataState{
				Annotations: annotationsFor(e).Fields,
			},
		}
	}
	links := map[string]*state.LinkState{}
	for _, link := range celerityLinks() {
		links[link] = &state.LinkState{LinkID: "link-" + link, Name: link, Status: core.LinkStatusCreated}
	}
	return &state.InstanceState{
		InstanceID:   "harness-instance",
		InstanceName: "orders-app",
		Status:       core.InstanceStatusDeployed,
		ResourceIDs:  resourceIDs,
		Resources:    resources,
		Links:        links,
	}
}
