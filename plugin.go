package panyl

type Plugin interface {
	IsPanylPlugin()
}

// PluginClean allows cleaning of a line.
// Change result.Line if you need to modify the line.
// You can set result.Metadata to allow other plugins to detect the change.
type PluginClean interface {
	Plugin
	Clean(result *Process) (bool, error)
}

// PluginMetadata allows extracting metadata from a line.
// Set result.Metadata with the detected data.
// You can also change result.Line if you need to remove the metadata from the line.
type PluginMetadata interface {
	Plugin
	ExtractMetadata(result *Process) (bool, error)
}

// PluginStructure allows extracting structure from a line, for example, JSON or XML.
// The full text must be a complete structure, partial match should not be supported.
// You should take in account the lines Metdatada/Data and apply them to result at your convenience.
type PluginStructure interface {
	Plugin
	ExtractStructure(lines ProcessLines, result *Process) (bool, error)
}

// PluginParse allows parsing data from a line, for example, an Apache log format, a Ruby log format, etc.
// The full text must be completely parsed, partial match should not be supported.
// You should take in account the lines Metdatada/Data and apply them to result at your convenience.
type PluginParse interface {
	Plugin
	ExtractParse(lines ProcessLines, result *Process) (bool, error)
}

// PluginSequence allows checking if 2 processes breaks a sequence, for example, if they belong to different
// applications, given it is possible to detect this.
type PluginSequence interface {
	Plugin
	BlockSequence(lastp, p *Process) bool
}

// PluginConsolidate allows to consolidate lines that couldn't be parsed by any plugin, like for example,
// multi-line Ruby error strings.
// The plugin should ALWAYS read lines from the top of the list, and set data in result about them.
// The topLines result states how many lines were processed, and they will be removed from future calls.
// The plugin can be called multiple times for the same set of lines, so don't try to detect more if you
// find a line that don't match, you will be called again after the unmatched line.
type PluginConsolidate interface {
	Plugin
	Consolidate(lines ProcessLines, result *Process) (_ bool, topLines int, _ error)
}

// PluginParseFormat is called for results that don't have Metadata_Format set, so it allows
// detecting some format from a raw structure (JSON or XML), for example, detecting the Apache log format from
// the parsed JSON data.
type PluginParseFormat interface {
	Plugin
	ParseFormat(result *Process) (bool, error)
}

// PluginCreate allows creating process entries that are not present in the log file.
// Use this to add custom log entries to the output.
// This is called after PluginPostProcess, and PluginPostProcess is also called for each item.
// Metadata_Created is set as true for items created by these functions.
type PluginCreate interface {
	Plugin
	CreateBefore(result *Process) ([]*Process, error)
	CreateAfter(result *Process) ([]*Process, error)
}

// PluginPostProcess is called right before the data is returned to the user, so it allows to do any final
// post-processing on the data.
// Order determines in which order post process plugins execute, lower execute first than higher.
// Use PostProcessOrder_Default as default. PostProcessOrder_First and PostProcessOrder_Last should be used
// as limits.
type PluginPostProcess interface {
	Plugin
	PostProcessOrder() int
	PostProcess(result *Process) (bool, error)
}
