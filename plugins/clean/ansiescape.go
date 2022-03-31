package clean

import (
	"github.com/RangelReale/panyl"
	"github.com/RangelReale/panyl/util"
)

var _ panyl.PluginClean = (*AnsiEscape)(nil)

// AnsiEscape implements PluginClean to remove ansi-escapes from the line
// it adds a Metadata_Clean metadata with value MetadataClean_AnsiEscape
type AnsiEscape struct {
}

func (c AnsiEscape) Clean(result *panyl.Process) (bool, error) {
	if ok, cl := util.AnsiEscapeString(result.Line); ok {
		result.Metadata.ListValueAdd(panyl.Metadata_Clean, panyl.MetadataClean_AnsiEscape)
		result.Line = cl
		return true, nil
	}
	return false, nil
}

func (c AnsiEscape) IsPanylPlugin() {}
