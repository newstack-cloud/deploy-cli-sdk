package diagutils

const defaultPluginOrg = "newstack-cloud"

// A mapping that provides the most likely organisation for a provider or transformer namespace
// to be able to better guide users with a full plugin ID for suggested actions.
var pluginOrgMappings = map[string]string{
	"aws":        defaultPluginOrg,
	"azure":      defaultPluginOrg,
	"gcloud":     defaultPluginOrg,
	"kubernetes": defaultPluginOrg,
	"celerity":   defaultPluginOrg,
}

func getOrgForPluginNamespace(pluginNamespace string) string {
	org, ok := pluginOrgMappings[pluginNamespace]
	if !ok {
		// When no mapping is found, use a placeholder organisation
		// that is to be replaced by the user when they carry out the suggested action.
		return "<organisation>"
	}
	return org
}
