package consolidate_test

import (
	"context"
	"testing"

	"github.com/RangelReale/panyl/v2"
	"github.com/RangelReale/panyl/v2/plugins/consolidate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJoinAllLines(t *testing.T) {
	lines := panyl.ItemLines{
		panyl.InitItem(panyl.WithInitLine("one"), panyl.WithInitCustom(func(item *panyl.Item) {
			item.Metadata[panyl.MetadataApplication] = "app"
			item.Data["a"] = 1
		})),
		panyl.InitItem(panyl.WithInitLine("two"), panyl.WithInitCustom(func(item *panyl.Item) {
			item.Metadata[panyl.MetadataApplication] = "other"
			item.Data["b"] = 2
		})),
		panyl.InitItem(panyl.WithInitLine("three")),
	}

	item := panyl.InitItem()
	ok, topLines, err := consolidate.JoinAllLines{}.Consolidate(context.Background(), lines, item)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, 3, topLines)
	assert.Equal(t, "one\ntwo\nthree", item.Line)
	// the first line wins for keys that exist in more than one line
	assert.Equal(t, "app", item.Metadata.StringValue(panyl.MetadataApplication))
	assert.Equal(t, panyl.MapValue{"a": 1, "b": 2}, item.Data)
}

func TestJoinAllLines_SingleLine(t *testing.T) {
	item := panyl.InitItem()
	ok, topLines, err := consolidate.JoinAllLines{}.Consolidate(context.Background(),
		panyl.ItemLines{panyl.InitItem(panyl.WithInitLine("one"))}, item)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, 1, topLines)
	assert.Equal(t, "one", item.Line)
}
