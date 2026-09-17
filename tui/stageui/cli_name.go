package stageui

// The CLI binary name used in user-facing hints. It defaults to "bluelink" and
// is overridden per binary via SetCLIName (the SDK sets it from
// CLIConfig.CLIName at command setup). A package-level value is safe here
// because each consuming CLI is a separate process.
//
// This mirrors the same mechanism in tui/driftui with a hint that names the wrong
// binary is worse than no hint, because it is a command the reader can copy and
// run only to be told it does not exist.
var cliName = "bluelink"

// SetCLIName overrides the CLI name used in stage hints. Empty values are
// ignored so the "bluelink" default is preserved.
func SetCLIName(name string) {
	if name != "" {
		cliName = name
	}
}
