package metadata

import (
	"github.com/RangelReale/panyl"
)

var _ panyl.PluginMetadata = (*ForceApplication)(nil)
var _ panyl.PluginSequence = (*ForceApplication)(nil)

// ForceApplication adds a Metadata_Application to the process metadata.
// It also blocks the sequence if the application changes.
type ForceApplication struct {
	Application string
}

func (m *ForceApplication) ExtractMetadata(result *panyl.Process) (bool, error) {
	result.Metadata[panyl.Metadata_Application] = m.Application
	return true, nil
}

func (m *ForceApplication) BlockSequence(lastp, p *panyl.Process) bool {
	// block sequence if application changed
	return lastp.Metadata.StringValue(panyl.Metadata_Application) != p.Metadata.StringValue(panyl.Metadata_Application)
}

func (m ForceApplication) IsPanylPlugin() {}
