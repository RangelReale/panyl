package clean

import (
	"github.com/RangelReale/panyl"
	"regexp"
)

var _ panyl.PluginClean = (*AnsiEscape)(nil)

// AnsiEscape implements PluginClean to remove ansi-escapes from the line
// it adds a Metadata_Clean metadata with value MetadataClean_AnsiEscape
type AnsiEscape struct {
}

// https://stackoverflow.com/a/33925425/784175
var cleanAnsiEscapeRE = regexp.MustCompile(`(\x9B|\x1B\[)[0-?]*[ -\/]*[@-~]`)

func (c AnsiEscape) Clean(result *panyl.Process) (bool, error) {
	count := 0
	ret := cleanAnsiEscapeRE.ReplaceAllStringFunc(result.Line, func(s string) string {
		count++
		return ""
	})
	if count > 0 {
		result.Metadata.ListValueAdd(panyl.Metadata_Clean, panyl.MetadataClean_AnsiEscape)
		result.Line = ret
		return true, nil
	}
	return false, nil
}

func (c AnsiEscape) IsPanylPlugin() {}
