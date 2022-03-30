package consolidate

import (
	"github.com/RangelReale/panyl"
)

var _ panyl.PluginConsolidate = (*JoinAllLines)(nil)

// JoinAllLines consolidates all consecutive non-parsed lines in a single result
type JoinAllLines struct {
}

func (j JoinAllLines) Consolidate(lines panyl.ProcessLines, result *panyl.Process) (_ bool, topLines int, _ error) {
	err := result.MergeLinesData(lines)
	if err != nil {
		return false, -1, err
	}
	result.Line = lines.Line()
	return true, len(lines), nil
}

func (j JoinAllLines) IsPanylPlugin() {}
