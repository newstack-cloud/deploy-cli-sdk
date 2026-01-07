package ui

// SelectBlueprintMsg is sent when a blueprint file has been selected.
type SelectBlueprintMsg struct {
	BlueprintFile string
	Source        string
}

// SelectBlueprintSourceMsg is sent when a blueprint source has been selected.
type SelectBlueprintSourceMsg struct {
	Source string
}

// ClearSelectedBlueprintMsg is sent to clear the current blueprint selection.
type ClearSelectedBlueprintMsg struct{}

// SelectBlueprintFileErrorMsg is sent when there's an error during blueprint selection.
type SelectBlueprintFileErrorMsg struct {
	Err error
}

// SelectBlueprintStartMsg is sent when the user makes a selection at the start screen.
type SelectBlueprintStartMsg struct {
	Selection string
}
