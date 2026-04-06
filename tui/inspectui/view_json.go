package inspectui

import (
	"github.com/newstack-cloud/deploy-cli-sdk/jsonout"
)

func (m *InspectModel) outputJSON() {
	if m.instanceState == nil {
		jsonout.WriteJSON(m.headlessWriter, nil)
		return
	}
	jsonout.WriteJSON(m.headlessWriter, m.instanceState)
}

func (m *InspectModel) outputJSONError(err error) {
	output := jsonout.NewErrorOutput(err)
	jsonout.WriteJSON(m.headlessWriter, output)
}
