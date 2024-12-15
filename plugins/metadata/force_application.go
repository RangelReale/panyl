package metadata

import (
	"context"

	"github.com/RangelReale/panyl/v2"
)

// ForceApplication adds a Metadata_Application to the process metadata if it isn't set already.
// It also blocks the sequence if the application changes.
type ForceApplication struct {
	Application string
}

var _ panyl.PluginMetadata = ForceApplication{}
var _ panyl.PluginSequence = ForceApplication{}

func (m ForceApplication) ExtractMetadata(ctx context.Context, item *panyl.Item) (bool, error) {
	if _, ok := item.Metadata[panyl.MetadataApplication]; !ok {
		item.Metadata[panyl.MetadataApplication] = m.Application
	}
	return true, nil
}

func (m ForceApplication) BlockSequence(ctx context.Context, lastp, item *panyl.Item) bool {
	// block sequence if application changed
	return lastp.Metadata.StringValue(panyl.MetadataApplication) != item.Metadata.StringValue(panyl.MetadataApplication)
}

func (m ForceApplication) IsPanylPlugin() {}
