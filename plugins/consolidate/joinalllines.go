package consolidate

import (
	"context"

	"github.com/RangelReale/panyl/v2"
)

// JoinAllLines consolidates all consecutive non-parsed lines in a single result
type JoinAllLines struct {
}

var _ panyl.PluginConsolidate = (*JoinAllLines)(nil)

func (j JoinAllLines) Consolidate(ctx context.Context, lines panyl.ItemLines, result *panyl.Item) (_ bool, topLines int, _ error) {
	err := result.MergeLinesData(lines)
	if err != nil {
		return false, -1, err
	}
	result.Line = lines.Line()
	return true, len(lines), nil
}

func (j JoinAllLines) IsPanylPlugin() {}
