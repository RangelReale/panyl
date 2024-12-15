package clean

import (
	"context"

	"github.com/RangelReale/panyl/v2"
	"github.com/RangelReale/panyl/v2/util"
)

// AnsiEscape implements PluginClean to remove ansi-escapes from the line
// it adds a Metadata_Clean metadata with value MetadataClean_AnsiEscape
type AnsiEscape struct {
}

var _ panyl.PluginClean = (*AnsiEscape)(nil)

func (c AnsiEscape) Clean(ctx context.Context, item *panyl.Item) (bool, error) {
	if ok, cl := util.AnsiEscapeString(item.Line); ok {
		item.Metadata.ListValueAdd(panyl.MetadataClean, panyl.MetadataCleanAnsiEscape)
		item.Line = cl
		return true, nil
	}
	return false, nil
}

func (c AnsiEscape) IsPanylPlugin() {}
