package consolidate

import (
	"context"

	"github.com/RangelReale/panyl"
)

// JoinAllLines consolidates all consecutive non-parsed lines in a single result
type JoinAllLines struct {
}

var _ panyl.PluginConsolidate = (*JoinAllLines)(nil)

func (j JoinAllLines) Consolidate(ctx context.Context, lines panyl.ProcessLines, result *panyl.Process) (_ bool, topLines int, _ error) {
	err := result.MergeLinesData(lines)
	if err != nil {
		return false, -1, err
	}
	result.Line = lines.Line()
	return true, len(lines), nil
}

func (j JoinAllLines) IsPanylPlugin() {}
