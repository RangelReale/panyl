package clean

import (
	"context"

	"github.com/RangelReale/panyl"
	"github.com/RangelReale/panyl/util"
)

// AnsiEscape implements PluginClean to remove ansi-escapes from the line
// it adds a Metadata_Clean metadata with value MetadataClean_AnsiEscape
type AnsiEscape struct {
}

var _ panyl.PluginClean = (*AnsiEscape)(nil)

func (c AnsiEscape) Clean(ctx context.Context, result *panyl.Process) (bool, error) {
	if ok, cl := util.AnsiEscapeString(result.Line); ok {
		result.Metadata.ListValueAdd(panyl.MetadataClean, panyl.MetadataCleanAnsiEscape)
		result.Line = cl
		return true, nil
	}
	return false, nil
}

func (c AnsiEscape) IsPanylPlugin() {}
