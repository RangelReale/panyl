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

var _ panyl.PluginMetadata = (*ForceApplication)(nil)
var _ panyl.PluginSequence = (*ForceApplication)(nil)

func (m *ForceApplication) ExtractMetadata(ctx context.Context, result *panyl.Item) (bool, error) {
	if _, ok := result.Metadata[panyl.MetadataApplication]; !ok {
		result.Metadata[panyl.MetadataApplication] = m.Application
	}
	return true, nil
}

func (m *ForceApplication) BlockSequence(ctx context.Context, lastp, item *panyl.Item) bool {
	// block sequence if application changed
	return lastp.Metadata.StringValue(panyl.MetadataApplication) != item.Metadata.StringValue(panyl.MetadataApplication)
}

func (m ForceApplication) IsPanylPlugin() {}
